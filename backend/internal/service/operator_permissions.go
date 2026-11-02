package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/operatoraccess"
)

const operatorPermissionsKeyPrefix = "operator_permissions:"

func operatorPermissionsKey(userID int64) string {
	return fmt.Sprintf("%s%d", operatorPermissionsKeyPrefix, userID)
}

// GetOperatorPermissions 读取运营用户被授予的权限；未配置时返回空集合（即无任何后台权限）。
func (s *SettingService) GetOperatorPermissions(ctx context.Context, userID int64) ([]string, error) {
	raw, err := s.settingRepo.GetValue(ctx, operatorPermissionsKey(userID))
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return []string{}, nil
		}
		return nil, err
	}
	var perms []string
	if raw == "" {
		return []string{}, nil
	}
	if err := json.Unmarshal([]byte(raw), &perms); err != nil {
		// 损坏的数据按无权限处理，避免误放行。
		return []string{}, nil
	}
	return operatoraccess.Sanitize(perms), nil
}

// SetOperatorPermissions 覆盖保存运营用户的权限，返回实际保存的（已校验）集合。
func (s *SettingService) SetOperatorPermissions(ctx context.Context, userID int64, perms []string) ([]string, error) {
	clean := operatoraccess.Sanitize(perms)
	data, err := json.Marshal(clean)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, operatorPermissionsKey(userID), string(data)); err != nil {
		return nil, err
	}
	return clean, nil
}
