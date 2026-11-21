package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"
)

// OperationsSiteAccount 是运营中心卡片里展示的账号摘要。
type OperationsSiteAccount struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

// OperationsSite 是按 base_url 去重后的一个上游站点。
type OperationsSite struct {
	Key            string                  `json:"key"`
	BaseURL        string                  `json:"base_url"`
	SiteURL        string                  `json:"site_url"`
	Host           string                  `json:"host"`
	AccountCount   int                     `json:"account_count"`
	Accounts       []OperationsSiteAccount `json:"accounts"`
	SourceAccount  *OperationsSiteAccount  `json:"source_account,omitempty"`
	Mode           string                  `json:"mode,omitempty"`
	Unit           string                  `json:"unit,omitempty"`
	PlanName       string                  `json:"plan_name,omitempty"`
	TotalQuota     *float64                `json:"total_quota,omitempty"`
	UsedQuota      *float64                `json:"used_quota,omitempty"`
	RemainingQuota *float64                `json:"remaining_quota,omitempty"`
	Unlimited      bool                    `json:"unlimited"`
	TodayCost      *float64                `json:"today_cost,omitempty"`
	UpdatedAt      *time.Time              `json:"updated_at,omitempty"`
	HasBalance     bool                    `json:"has_balance"`
}

// OperationsSiteKey 把 base_url 归一化为去重键：小写 scheme/host，去掉末尾 "/" 与 "/v1"。
func OperationsSiteKey(rawBaseURL string) (key, siteURL, host string) {
	raw := strings.TrimSpace(rawBaseURL)
	if raw == "" {
		return "", "", ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.ToLower(strings.TrimRight(strings.TrimSpace(rawBaseURL), "/")), "", ""
	}
	scheme := strings.ToLower(u.Scheme)
	host = strings.ToLower(u.Host)
	path := strings.TrimRight(u.Path, "/")
	for _, suffix := range []string{"/v1", "/api"} {
		path = strings.TrimSuffix(path, suffix)
	}
	path = strings.TrimRight(path, "/")
	siteURL = scheme + "://" + host + path
	return siteURL, siteURL, host
}

// BuildOperationsSites 把 API Key / 上游账号按 base_url 聚合，并读取已缓存的上游余额快照。
// 同站点多账号时取最近一次更新的快照作为站点额度。
func BuildOperationsSites(accounts []Account) []OperationsSite {
	type siteAgg struct {
		site     OperationsSite
		snapshot *UpstreamBalanceSnapshot
	}
	order := make([]string, 0)
	sites := make(map[string]*siteAgg)

	for i := range accounts {
		account := &accounts[i]
		if account.Type != AccountTypeAPIKey && account.Type != AccountTypeUpstream {
			continue
		}
		key, siteURL, host := OperationsSiteKey(upstreamBalanceBaseURL(account))
		if key == "" {
			continue
		}
		summary := OperationsSiteAccount{
			ID:       account.ID,
			Name:     account.Name,
			Platform: account.Platform,
			Type:     account.Type,
			Status:   account.Status,
		}
		agg, ok := sites[key]
		if !ok {
			agg = &siteAgg{site: OperationsSite{
				Key:     key,
				BaseURL: strings.TrimSpace(upstreamBalanceBaseURL(account)),
				SiteURL: siteURL,
				Host:    host,
			}}
			sites[key] = agg
			order = append(order, key)
		}
		agg.site.Accounts = append(agg.site.Accounts, summary)
		agg.site.AccountCount++

		snapshot := operationsSnapshotFromAccount(account)
		if snapshot == nil {
			continue
		}
		if agg.snapshot == nil || snapshot.UpdatedAt.After(agg.snapshot.UpdatedAt) {
			agg.snapshot = snapshot
			src := summary
			agg.site.SourceAccount = &src
		}
	}

	result := make([]OperationsSite, 0, len(order))
	for _, key := range order {
		agg := sites[key]
		site := agg.site
		applyOperationsSnapshot(&site, agg.snapshot)
		result = append(result, site)
	}

	// 有余额数据的排前面，按剩余额度升序（快用完的优先关注），无数据的按 host 排序。
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.HasBalance != b.HasBalance {
			return a.HasBalance
		}
		if a.HasBalance && a.RemainingQuota != nil && b.RemainingQuota != nil && *a.RemainingQuota != *b.RemainingQuota {
			return *a.RemainingQuota < *b.RemainingQuota
		}
		return a.Host < b.Host
	})
	return result
}

func operationsSnapshotFromAccount(account *Account) *UpstreamBalanceSnapshot {
	if account == nil || account.Extra == nil {
		return nil
	}
	raw, ok := account.Extra["upstream_balance"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *UpstreamBalanceSnapshot:
		return v
	case UpstreamBalanceSnapshot:
		return &v
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var snapshot UpstreamBalanceSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil
	}
	return &snapshot
}

func applyOperationsSnapshot(site *OperationsSite, snapshot *UpstreamBalanceSnapshot) {
	if site == nil || snapshot == nil {
		return
	}
	site.Mode = snapshot.Mode
	site.Unit = snapshot.Unit
	site.PlanName = snapshot.PlanName
	site.TodayCost = finiteOrNil(snapshot.TodayCost)
	if !snapshot.UpdatedAt.IsZero() {
		t := snapshot.UpdatedAt
		site.UpdatedAt = &t
	}
	if snapshot.QuotaUnlimited != nil && *snapshot.QuotaUnlimited {
		site.Unlimited = true
	}
	if remaining, ok := upstreamBalanceSnapshotValue(snapshot); ok {
		site.RemainingQuota = &remaining
	}
	site.UsedQuota = finiteOrNil(snapshot.QuotaUsed)
	if site.UsedQuota == nil {
		site.UsedQuota = finiteOrNil(snapshot.TotalCost)
	}
	site.TotalQuota = finiteOrNil(snapshot.QuotaLimit)
	if site.TotalQuota == nil && site.RemainingQuota != nil && site.UsedQuota != nil {
		total := *site.RemainingQuota + *site.UsedQuota
		site.TotalQuota = &total
	}
	site.HasBalance = site.RemainingQuota != nil || site.Unlimited
}

func finiteOrNil(v *float64) *float64 {
	if v == nil || math.IsNaN(*v) || math.IsInf(*v, 0) {
		return nil
	}
	out := *v
	return &out
}

// AccountRequestRankingItem 是某个账号在时间范围内的请求统计。
type AccountRequestRankingItem struct {
	AccountID       int64      `json:"account_id"`
	Name            string     `json:"name"`
	Platform        string     `json:"platform"`
	Type            string     `json:"type"`
	Status          string     `json:"status"`
	Deleted         bool       `json:"deleted"`
	BaseURL         string     `json:"base_url,omitempty"`
	Requests        int64      `json:"requests"`
	InputTokens     int64      `json:"input_tokens"`
	OutputTokens    int64      `json:"output_tokens"`
	CacheTokens     int64      `json:"cache_tokens"`
	Tokens          int64      `json:"tokens"`
	AccountCost     float64    `json:"account_cost"`
	UserCost        float64    `json:"user_cost"`
	AvgDurationMs   float64    `json:"avg_duration_ms"`
	AvgFirstTokenMs float64    `json:"avg_first_token_ms"`
	Users           int64      `json:"users"`
	Models          int64      `json:"models"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	Errors          int64      `json:"errors"`
	RateLimited     int64      `json:"rate_limited"`
}

// AccountRequestRankingResponse 是账号请求排行（按请求数降序）及全局合计。
type AccountRequestRankingResponse struct {
	Items            []AccountRequestRankingItem `json:"items"`
	TotalRequests    int64                       `json:"total_requests"`
	TotalTokens      int64                       `json:"total_tokens"`
	TotalAccountCost float64                     `json:"total_account_cost"`
	TotalUserCost    float64                     `json:"total_user_cost"`
	TotalAccounts    int64                       `json:"total_accounts"`
	TotalErrors      int64                       `json:"total_errors"`
	TotalSites       int64                       `json:"total_sites,omitempty"`
	GroupBy          string                      `json:"group_by"`
	Sites            []SiteRequestRankingItem    `json:"sites,omitempty"`
}

type accountRequestRanker interface {
	GetAccountRequestRanking(ctx context.Context, startTime, endTime time.Time) (*AccountRequestRankingResponse, error)
}

// SiteRequestRankingItem 是按 base_url 合并后的站点请求统计。
// 没有 base_url 的账号（如 OAuth 账号）各自单独成行，key 为 "account:<id>"。
type SiteRequestRankingItem struct {
	Key             string                  `json:"key"`
	BaseURL         string                  `json:"base_url"`
	SiteURL         string                  `json:"site_url"`
	Host            string                  `json:"host"`
	AccountCount    int                     `json:"account_count"`
	Accounts        []OperationsSiteAccount `json:"accounts"`
	Requests        int64                   `json:"requests"`
	InputTokens     int64                   `json:"input_tokens"`
	OutputTokens    int64                   `json:"output_tokens"`
	CacheTokens     int64                   `json:"cache_tokens"`
	Tokens          int64                   `json:"tokens"`
	AccountCost     float64                 `json:"account_cost"`
	UserCost        float64                 `json:"user_cost"`
	AvgDurationMs   float64                 `json:"avg_duration_ms"`
	AvgFirstTokenMs float64                 `json:"avg_first_token_ms"`
	LastUsedAt      *time.Time              `json:"last_used_at,omitempty"`
	Errors          int64                   `json:"errors"`
	RateLimited     int64                   `json:"rate_limited"`
}

// GetAccountRequestRanking 返回时间范围内的请求排行（按请求数从高到低）。
// groupBySite 为 true 时按 base_url 去重合并为站点行（Sites），否则返回账号行（Items）。
func (s *DashboardService) GetAccountRequestRanking(ctx context.Context, startTime, endTime time.Time, limit int, groupBySite bool) (*AccountRequestRankingResponse, error) {
	ranker, ok := s.usageRepo.(accountRequestRanker)
	if !ok {
		return nil, fmt.Errorf("account request ranking not supported by usage repository")
	}
	resp, err := ranker.GetAccountRequestRanking(ctx, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get account request ranking: %w", err)
	}
	if limit <= 0 {
		limit = 100
	}
	if groupBySite {
		resp.Sites = GroupRequestRankingBySite(resp.Items)
		resp.TotalSites = int64(len(resp.Sites))
		if len(resp.Sites) > limit {
			resp.Sites = resp.Sites[:limit]
		}
		resp.Items = nil
		resp.GroupBy = "site"
		return resp, nil
	}
	if len(resp.Items) > limit {
		resp.Items = resp.Items[:limit]
	}
	resp.GroupBy = "account"
	return resp, nil
}

// GroupRequestRankingBySite 按归一化的 base_url 合并账号请求统计，结果按请求数降序。
func GroupRequestRankingBySite(items []AccountRequestRankingItem) []SiteRequestRankingItem {
	type agg struct {
		site          SiteRequestRankingItem
		durationSum   float64
		firstTokenSum float64
		firstTokenReq int64
	}
	order := make([]string, 0)
	groups := make(map[string]*agg)
	for _, item := range items {
		key, siteURL, host := OperationsSiteKey(item.BaseURL)
		if key == "" {
			key = fmt.Sprintf("account:%d", item.AccountID)
		}
		g, ok := groups[key]
		if !ok {
			g = &agg{site: SiteRequestRankingItem{
				Key:     key,
				BaseURL: item.BaseURL,
				SiteURL: siteURL,
				Host:    host,
			}}
			groups[key] = g
			order = append(order, key)
		}
		st := &g.site
		st.AccountCount++
		st.Accounts = append(st.Accounts, OperationsSiteAccount{
			ID: item.AccountID, Name: item.Name, Platform: item.Platform, Type: item.Type, Status: item.Status,
		})
		st.Requests += item.Requests
		st.InputTokens += item.InputTokens
		st.OutputTokens += item.OutputTokens
		st.CacheTokens += item.CacheTokens
		st.Tokens += item.Tokens
		st.AccountCost += item.AccountCost
		st.UserCost += item.UserCost
		st.Errors += item.Errors
		st.RateLimited += item.RateLimited
		g.durationSum += item.AvgDurationMs * float64(item.Requests)
		if item.AvgFirstTokenMs > 0 {
			g.firstTokenSum += item.AvgFirstTokenMs * float64(item.Requests)
			g.firstTokenReq += item.Requests
		}
		if item.LastUsedAt != nil && (st.LastUsedAt == nil || item.LastUsedAt.After(*st.LastUsedAt)) {
			t := *item.LastUsedAt
			st.LastUsedAt = &t
		}
	}
	result := make([]SiteRequestRankingItem, 0, len(order))
	for _, key := range order {
		g := groups[key]
		if g.site.Requests > 0 {
			g.site.AvgDurationMs = g.durationSum / float64(g.site.Requests)
		}
		if g.firstTokenReq > 0 {
			g.site.AvgFirstTokenMs = g.firstTokenSum / float64(g.firstTokenReq)
		}
		// 站点内账号按请求数降序（输入已按请求数降序，保持稳定即可）
		result = append(result, g.site)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Requests != result[j].Requests {
			return result[i].Requests > result[j].Requests
		}
		return result[i].Tokens > result[j].Tokens
	})
	return result
}
