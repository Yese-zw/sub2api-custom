package service

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// SchedulingCenterResetOptions 指定初始化分组调度时要清理的内容。
type SchedulingCenterResetOptions struct {
	ClearPool     bool `json:"clear_pool"`     // 清空调度池：所有成员移回账号池
	ClearStates   bool `json:"clear_states"`   // 清除冷却 / 恢复中 / 可入池 / 手动移出状态
	ClearEvents   bool `json:"clear_events"`   // 清空调度日志
	ClearRecords  bool `json:"clear_records"`  // 清空该分组托管检测计划的检测记录
	ResetSettings bool `json:"reset_settings"` // 规则参数恢复默认（保留调度开关、检测频率、检测模型）
}

func (o SchedulingCenterResetOptions) any() bool {
	return o.ClearPool || o.ClearStates || o.ClearEvents || o.ClearRecords || o.ResetSettings
}

// SchedulingCenterResetResult 是初始化的结果（各项删除的行数）。
type SchedulingCenterResetResult struct {
	PoolMembers   int64 `json:"pool_members"`
	States        int64 `json:"states"`
	Events        int64 `json:"events"`
	Records       int64 `json:"records"`
	SettingsReset bool  `json:"settings_reset"`
}

var (
	ErrSchedulingCenterResetNothing = errors.New("select at least one item to reset")
	ErrSchedulingCenterBusy         = errors.New("scheduling center is evaluating, please retry in a moment")
)

const schedulingCenterResetLockWait = 15 * time.Second

// ResetGroup 初始化分组的调度中心数据。与后台评估共用同一把锁，避免评估中途被清空。
// 检测记录按账号的托管计划清理：账号属于多个开启调度的分组时，其计划归属于分组 ID 最小的分组。
func (s *SchedulingCenterService) ResetGroup(ctx context.Context, groupID int64, opts SchedulingCenterResetOptions) (*SchedulingCenterResetResult, error) {
	if !opts.any() {
		return nil, ErrSchedulingCenterResetNothing
	}
	result := &SchedulingCenterResetResult{}
	defer s.gate.Notify()
	err := s.withLockWait(ctx, "scheduling_center:evaluate_rules", schedulingCenterResetLockWait, func() error {
		counts, err := s.repo.ResetGroupData(ctx, groupID, opts)
		if err != nil {
			return err
		}
		*result = *counts
		if opts.ResetSettings {
			current, err := s.GetGroupSettings(ctx, groupID)
			if err != nil {
				return err
			}
			def := defaultSchedulingCenterGroupSettings(groupID)
			def.Enabled, def.CronExpression, def.ModelID = current.Enabled, current.CronExpression, current.ModelID
			def.EnforceEnabled = current.EnforceEnabled // 生效开关影响真实调度，初始化不改动
			if _, err := s.repo.UpsertGroupSettings(ctx, def); err != nil {
				return err
			}
			result.SettingsReset = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	logger.LegacyPrintf("service.scheduling_center",
		"[SchedulingCenter] group=%d reset: pool=%d states=%d events=%d records=%d settings=%v",
		groupID, result.PoolMembers, result.States, result.Events, result.Records, result.SettingsReset)
	if opts.ResetSettings {
		// 保留条数可能变化，同步到托管计划；失败时后台任务会补齐
		if err := s.SyncPlans(ctx); err != nil {
			logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] sync plans after reset failed: %v", err)
		}
	}
	return result, nil
}

// withLockWait 与 withLock 相同，但拿不到锁时在 wait 时间内重试，而不是直接跳过。
func (s *SchedulingCenterService) withLockWait(ctx context.Context, key string, wait time.Duration, fn func() error) error {
	if s.db == nil {
		return fn()
	}
	deadline := time.Now().Add(wait)
	for {
		release, acquired, err := tryAcquireDBAdvisoryLockWithError(ctx, s.db, hashAdvisoryLockID(key))
		if err != nil {
			return err
		}
		if acquired {
			defer release()
			return fn()
		}
		if time.Now().After(deadline) {
			return ErrSchedulingCenterBusy
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}
