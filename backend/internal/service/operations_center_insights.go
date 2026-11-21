package service

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// AccountConsumption 是账号近 24 小时 / 7 天的账号口径消耗。
type AccountConsumption struct {
	AccountID   int64     `json:"account_id"`
	Requests24h int64     `json:"requests_24h"`
	Cost24h     float64   `json:"cost_24h"`
	Requests7d  int64     `json:"requests_7d"`
	Cost7d      float64   `json:"cost_7d"`
	FirstUsedAt time.Time `json:"first_used_at"`
}

// AccountTrendPoint 是某个时间桶内某账号的聚合数据。
type AccountTrendPoint struct {
	Bucket      string
	AccountID   int64
	Name        string
	BaseURL     string
	Requests    int64
	AccountCost float64
	Tokens      int64
	Errors      int64
}

// SiteTrendSeries 是一个站点（或"其他"）在各时间桶上的数据，数组下标与 Buckets 对齐。
type SiteTrendSeries struct {
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	SiteURL      string    `json:"site_url,omitempty"`
	AccountCount int       `json:"account_count"`
	Requests     []int64   `json:"requests"`
	AccountCost  []float64 `json:"account_cost"`
	Tokens       []int64   `json:"tokens"`
	Errors       []int64   `json:"errors"`
	TotalReq     int64     `json:"total_requests"`
	TotalCost    float64   `json:"total_account_cost"`
	TotalErrors  int64     `json:"total_errors"`
}

// SiteTrendResponse 是站点趋势：前 N 个站点 + 其他，以及全量合计。
type SiteTrendResponse struct {
	Granularity string            `json:"granularity"`
	Buckets     []string          `json:"buckets"`
	Series      []SiteTrendSeries `json:"series"`
	Total       SiteTrendSeries   `json:"total"`
}

type operationsInsightsRepo interface {
	GetAccountConsumption(ctx context.Context, now time.Time) ([]AccountConsumption, error)
	GetAccountTrendPoints(ctx context.Context, startTime, endTime time.Time, granularity, loc string) ([]AccountTrendPoint, error)
	GetRequestHeatmap(ctx context.Context, startTime, endTime time.Time, loc string) ([]HeatmapCell, error)
}

// HeatmapCell 是星期 × 小时热力图的一个格子（Weekday：1=周一 … 7=周日）。
type HeatmapCell struct {
	Weekday     int     `json:"weekday"`
	Hour        int     `json:"hour"`
	Requests    int64   `json:"requests"`
	AccountCost float64 `json:"account_cost"`
	Errors      int64   `json:"errors"`
}

// GetRequestHeatmap 返回时间范围内按星期 × 小时聚合的请求热力图（时区无效时回退 UTC）。
func (s *DashboardService) GetRequestHeatmap(ctx context.Context, startTime, endTime time.Time, loc string) ([]HeatmapCell, error) {
	repo, err := s.operationsRepo()
	if err != nil {
		return nil, err
	}
	if _, err := time.LoadLocation(loc); err != nil || loc == "" {
		loc = "UTC"
	}
	cells, err := repo.GetRequestHeatmap(ctx, startTime, endTime, loc)
	if err != nil {
		return nil, fmt.Errorf("get request heatmap: %w", err)
	}
	return cells, nil
}

func (s *DashboardService) operationsRepo() (operationsInsightsRepo, error) {
	repo, ok := s.usageRepo.(operationsInsightsRepo)
	if !ok {
		return nil, fmt.Errorf("operations insights not supported by usage repository")
	}
	return repo, nil
}

// GetAccountConsumption 返回各账号近 24 小时 / 7 天消耗。
func (s *DashboardService) GetAccountConsumption(ctx context.Context) ([]AccountConsumption, error) {
	repo, err := s.operationsRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetAccountConsumption(ctx, time.Now())
}

// GetSiteTrend 返回按 base_url 合并的站点趋势，取请求数前 topN 的站点，其余合并为"其他"。
func (s *DashboardService) GetSiteTrend(ctx context.Context, startTime, endTime time.Time, granularity, loc string, topN int) (*SiteTrendResponse, error) {
	repo, err := s.operationsRepo()
	if err != nil {
		return nil, err
	}
	if granularity != "day" {
		granularity = "hour"
	}
	location, err := time.LoadLocation(loc)
	if err != nil || loc == "" {
		loc, location = "UTC", time.UTC
	}
	points, err := repo.GetAccountTrendPoints(ctx, startTime, endTime, granularity, loc)
	if err != nil {
		return nil, fmt.Errorf("get site trend: %w", err)
	}
	return BuildSiteTrend(points, startTime, endTime, granularity, location, topN), nil
}

// OperationsOtherSiteKey 是趋势中"其他站点"系列的 key。
const OperationsOtherSiteKey = "__other__"

// BuildSiteTrend 把账号趋势点按站点合并，补齐空桶，并截取前 topN 个站点。
func BuildSiteTrend(points []AccountTrendPoint, startTime, endTime time.Time, granularity string, loc *time.Location, topN int) *SiteTrendResponse {
	if topN <= 0 {
		topN = 8
	}
	if loc == nil {
		loc = time.UTC
	}
	const layout = "2006-01-02T15:04:05"

	// 生成连续时间桶
	buckets := make([]string, 0)
	index := make(map[string]int)
	cur := truncateBucket(startTime.In(loc), granularity)
	for i := 0; cur.Before(endTime) && i < 24*400; i++ {
		key := cur.Format(layout)
		index[key] = len(buckets)
		buckets = append(buckets, key)
		if granularity == "day" {
			cur = cur.AddDate(0, 0, 1)
		} else {
			cur = cur.Add(time.Hour)
		}
	}
	newSeries := func(key, label, siteURL string) *SiteTrendSeries {
		return &SiteTrendSeries{
			Key: key, Label: label, SiteURL: siteURL,
			Requests:    make([]int64, len(buckets)),
			AccountCost: make([]float64, len(buckets)),
			Tokens:      make([]int64, len(buckets)),
			Errors:      make([]int64, len(buckets)),
		}
	}
	add := func(sr *SiteTrendSeries, i int, p AccountTrendPoint) {
		sr.Requests[i] += p.Requests
		sr.AccountCost[i] += p.AccountCost
		sr.Tokens[i] += p.Tokens
		sr.Errors[i] += p.Errors
		sr.TotalReq += p.Requests
		sr.TotalCost += p.AccountCost
		sr.TotalErrors += p.Errors
	}

	sites := make(map[string]*SiteTrendSeries)
	accounts := make(map[string]map[int64]struct{})
	total := newSeries("__total__", "", "")
	for _, p := range points {
		i, ok := index[p.Bucket]
		if !ok {
			continue
		}
		key, siteURL, host := OperationsSiteKey(p.BaseURL)
		label := host
		if key == "" {
			key = fmt.Sprintf("account:%d", p.AccountID)
			label = p.Name
			if label == "" {
				label = fmt.Sprintf("#%d", p.AccountID)
			}
		}
		sr, ok := sites[key]
		if !ok {
			sr = newSeries(key, label, siteURL)
			sites[key] = sr
			accounts[key] = make(map[int64]struct{})
		}
		accounts[key][p.AccountID] = struct{}{}
		add(sr, i, p)
		add(total, i, p)
	}

	all := make([]*SiteTrendSeries, 0, len(sites))
	for key, sr := range sites {
		sr.AccountCount = len(accounts[key])
		all = append(all, sr)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].TotalReq != all[j].TotalReq {
			return all[i].TotalReq > all[j].TotalReq
		}
		return all[i].Key < all[j].Key
	})

	series := make([]SiteTrendSeries, 0, topN+1)
	var other *SiteTrendSeries
	for i, sr := range all {
		if i < topN {
			series = append(series, *sr)
			continue
		}
		if other == nil {
			other = newSeries(OperationsOtherSiteKey, "", "")
		}
		other.AccountCount += sr.AccountCount
		for b := range buckets {
			add(other, b, AccountTrendPoint{Requests: sr.Requests[b], AccountCost: sr.AccountCost[b], Tokens: sr.Tokens[b], Errors: sr.Errors[b]})
		}
	}
	if other != nil {
		series = append(series, *other)
	}
	return &SiteTrendResponse{Granularity: granularity, Buckets: buckets, Series: series, Total: *total}
}

func truncateBucket(t time.Time, granularity string) time.Time {
	if granularity == "day" {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}
