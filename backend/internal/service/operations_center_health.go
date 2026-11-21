package service

import (
	"sort"
	"time"
)

// 账号健康问题类型，按严重程度从高到低。
const (
	HealthIssueError             = "error"
	HealthIssueExpired           = "expired"
	HealthIssueQuotaExceeded     = "quota_exceeded"
	HealthIssueRateLimited       = "rate_limited"
	HealthIssueOverloaded        = "overloaded"
	HealthIssueTempUnschedulable = "temp_unschedulable"
	HealthIssueExpiringSoon      = "expiring_soon"
	HealthIssuePaused            = "paused"
)

var healthIssueSeverity = map[string]int{
	HealthIssueError:             0,
	HealthIssueExpired:           1,
	HealthIssueQuotaExceeded:     2,
	HealthIssueRateLimited:       3,
	HealthIssueOverloaded:        4,
	HealthIssueTempUnschedulable: 5,
	HealthIssueExpiringSoon:      6,
	HealthIssuePaused:            7,
}

const (
	operationsIdleThreshold     = 7 * 24 * time.Hour
	operationsExpiringThreshold = 3 * 24 * time.Hour
)

// HealthAccount 是健康页中的账号摘要。
type HealthAccount struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Platform   string     `json:"platform"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	BaseURL    string     `json:"base_url,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// HealthIssue 是一个需要关注的账号问题。
type HealthIssue struct {
	Account HealthAccount `json:"account"`
	Kind    string        `json:"kind"`
	Detail  string        `json:"detail,omitempty"`
	Until   *time.Time    `json:"until,omitempty"`
}

// HealthPlatformStat 是某平台的账号状态分布。
type HealthPlatformStat struct {
	Platform    string `json:"platform"`
	Total       int    `json:"total"`
	Schedulable int    `json:"schedulable"`
	Issues      int    `json:"issues"`
	Idle        int    `json:"idle"`
}

// AccountHealthSummary 汇总所有账号的健康状况。
type AccountHealthSummary struct {
	Total             int `json:"total"`
	Active            int `json:"active"`
	Disabled          int `json:"disabled"`
	Error             int `json:"error"`
	Schedulable       int `json:"schedulable"`
	RateLimited       int `json:"rate_limited"`
	Overloaded        int `json:"overloaded"`
	TempUnschedulable int `json:"temp_unschedulable"`
	QuotaExceeded     int `json:"quota_exceeded"`
	Expired           int `json:"expired"`
	ExpiringSoon      int `json:"expiring_soon"`
	Paused            int `json:"paused"`
	Idle              int `json:"idle"`
}

// AccountHealthResponse 是账号健康页数据。
type AccountHealthResponse struct {
	Summary       AccountHealthSummary `json:"summary"`
	Platforms     []HealthPlatformStat `json:"platforms"`
	Issues        []HealthIssue        `json:"issues"`
	Idle          []HealthAccount      `json:"idle"`
	IdleThreshold int64                `json:"idle_threshold_hours"`
	GeneratedAt   time.Time            `json:"generated_at"`
}

// BuildAccountHealth 根据账号当前状态计算健康汇总、问题列表与闲置账号。
// 闲置：启用且可调度、创建超过阈值、但阈值时间内没有被使用过（last_used_at）。
func BuildAccountHealth(accounts []Account, now time.Time) *AccountHealthResponse {
	resp := &AccountHealthResponse{
		Platforms:     make([]HealthPlatformStat, 0),
		Issues:        make([]HealthIssue, 0),
		Idle:          make([]HealthAccount, 0),
		IdleThreshold: int64(operationsIdleThreshold / time.Hour),
		GeneratedAt:   now,
	}
	platforms := make(map[string]*HealthPlatformStat)
	sum := &resp.Summary

	for i := range accounts {
		a := &accounts[i]
		ha := HealthAccount{
			ID: a.ID, Name: a.Name, Platform: a.Platform, Type: a.Type, Status: a.Status,
			BaseURL: upstreamBalanceBaseURL(a), LastUsedAt: a.LastUsedAt, CreatedAt: a.CreatedAt,
		}
		ps, ok := platforms[a.Platform]
		if !ok {
			ps = &HealthPlatformStat{Platform: a.Platform}
			platforms[a.Platform] = ps
		}
		ps.Total++
		sum.Total++

		switch a.Status {
		case StatusActive:
			sum.Active++
		case StatusDisabled:
			sum.Disabled++
		case StatusError:
			sum.Error++
		}

		issues := make([]HealthIssue, 0, 2)
		addIssue := func(kind, detail string, until *time.Time) {
			issues = append(issues, HealthIssue{Account: ha, Kind: kind, Detail: detail, Until: until})
		}
		active := a.Status == StatusActive
		if a.Status == StatusError {
			addIssue(HealthIssueError, a.ErrorMessage, nil)
		}
		if active {
			if a.AutoPauseOnExpired && a.ExpiresAt != nil {
				if !now.Before(*a.ExpiresAt) {
					sum.Expired++
					addIssue(HealthIssueExpired, "", a.ExpiresAt)
				} else if a.ExpiresAt.Sub(now) <= operationsExpiringThreshold {
					sum.ExpiringSoon++
					addIssue(HealthIssueExpiringSoon, "", a.ExpiresAt)
				}
			}
			if a.SupportsAccountQuota() && a.IsQuotaExceeded() {
				sum.QuotaExceeded++
				addIssue(HealthIssueQuotaExceeded, "", nil)
			}
			if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
				sum.RateLimited++
				addIssue(HealthIssueRateLimited, "", a.RateLimitResetAt)
			}
			if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
				sum.Overloaded++
				addIssue(HealthIssueOverloaded, "", a.OverloadUntil)
			}
			if a.TempUnschedulableUntil != nil && now.Before(*a.TempUnschedulableUntil) {
				sum.TempUnschedulable++
				addIssue(HealthIssueTempUnschedulable, a.TempUnschedulableReason, a.TempUnschedulableUntil)
			}
			if !a.Schedulable {
				sum.Paused++
				addIssue(HealthIssuePaused, "", nil)
			}
		}
		if len(issues) > 0 {
			ps.Issues++
			resp.Issues = append(resp.Issues, issues...)
		}

		if a.IsSchedulable() {
			sum.Schedulable++
			ps.Schedulable++
			olderThanThreshold := now.Sub(a.CreatedAt) >= operationsIdleThreshold
			unused := a.LastUsedAt == nil || now.Sub(*a.LastUsedAt) >= operationsIdleThreshold
			if olderThanThreshold && unused {
				sum.Idle++
				ps.Idle++
				resp.Idle = append(resp.Idle, ha)
			}
		}
	}

	for _, ps := range platforms {
		resp.Platforms = append(resp.Platforms, *ps)
	}
	sort.Slice(resp.Platforms, func(i, j int) bool {
		if resp.Platforms[i].Total != resp.Platforms[j].Total {
			return resp.Platforms[i].Total > resp.Platforms[j].Total
		}
		return resp.Platforms[i].Platform < resp.Platforms[j].Platform
	})
	sort.SliceStable(resp.Issues, func(i, j int) bool {
		si, sj := healthIssueSeverity[resp.Issues[i].Kind], healthIssueSeverity[resp.Issues[j].Kind]
		if si != sj {
			return si < sj
		}
		return resp.Issues[i].Account.ID < resp.Issues[j].Account.ID
	})
	// 闲置账号：从未使用的排最前，其余按最后使用时间从早到晚
	sort.SliceStable(resp.Idle, func(i, j int) bool {
		a, b := resp.Idle[i].LastUsedAt, resp.Idle[j].LastUsedAt
		if (a == nil) != (b == nil) {
			return a == nil
		}
		if a != nil && b != nil && !a.Equal(*b) {
			return a.Before(*b)
		}
		return resp.Idle[i].ID < resp.Idle[j].ID
	})
	return resp
}
