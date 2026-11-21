package service

import (
	"testing"
	"time"
)

func TestOperationsSiteKeyNormalizes(t *testing.T) {
	cases := map[string]string{
		"https://API.Example.com/v1/": "https://api.example.com",
		"https://api.example.com":     "https://api.example.com",
		"api.example.com/v1":          "https://api.example.com",
		"https://x.com/proxy/v1":      "https://x.com/proxy",
	}
	for in, want := range cases {
		if got, _, _ := OperationsSiteKey(in); got != want {
			t.Fatalf("OperationsSiteKey(%q)=%q want %q", in, got, want)
		}
	}
}

func TestBuildOperationsSitesDedupesByBaseURL(t *testing.T) {
	older := time.Now().Add(-time.Hour).UTC()
	newer := time.Now().UTC()
	accounts := []Account{
		{ID: 1, Name: "a", Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://a.com/v1"},
			Extra: map[string]any{"upstream_balance": map[string]any{"remaining": 5.0, "quota_used": 5.0, "unit": "USD", "updated_at": older.Format(time.RFC3339Nano)}}},
		{ID: 2, Name: "b", Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://A.com"},
			Extra: map[string]any{"upstream_balance": map[string]any{"remaining": 3.0, "quota_limit": 10.0, "unit": "USD", "updated_at": newer.Format(time.RFC3339Nano)}}},
		{ID: 3, Name: "c", Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://b.com"}},
		{ID: 4, Name: "oauth", Type: "oauth", Credentials: map[string]any{"base_url": "https://c.com"}},
	}
	sites := BuildOperationsSites(accounts)
	if len(sites) != 2 {
		t.Fatalf("want 2 sites, got %d", len(sites))
	}
	a := sites[0]
	if a.Host != "a.com" || a.AccountCount != 2 || a.SourceAccount == nil || a.SourceAccount.ID != 2 {
		t.Fatalf("unexpected site a: %+v", a)
	}
	if a.RemainingQuota == nil || *a.RemainingQuota != 3 || a.TotalQuota == nil || *a.TotalQuota != 10 {
		t.Fatalf("unexpected quota: %+v", a)
	}
	if sites[1].HasBalance || sites[1].Host != "b.com" {
		t.Fatalf("unexpected site b: %+v", sites[1])
	}
}

func TestGroupRequestRankingBySite(t *testing.T) {
	items := []AccountRequestRankingItem{
		{AccountID: 1, BaseURL: "https://a.com/v1", Requests: 10, Tokens: 100, AvgDurationMs: 100, AvgFirstTokenMs: 50},
		{AccountID: 2, BaseURL: "https://b.com", Requests: 15, Tokens: 10},
		{AccountID: 3, BaseURL: "https://A.com", Requests: 10, Tokens: 100, AvgDurationMs: 300},
		{AccountID: 4, Requests: 1},
	}
	sites := GroupRequestRankingBySite(items)
	if len(sites) != 3 {
		t.Fatalf("want 3 rows, got %d", len(sites))
	}
	if sites[0].Host != "a.com" || sites[0].Requests != 20 || sites[0].AccountCount != 2 {
		t.Fatalf("unexpected first row: %+v", sites[0])
	}
	if sites[0].AvgDurationMs != 200 || sites[0].AvgFirstTokenMs != 50 {
		t.Fatalf("unexpected averages: %+v", sites[0])
	}
	if sites[1].Host != "b.com" || sites[2].Key != "account:4" {
		t.Fatalf("unexpected order: %+v", sites)
	}
}

func TestBuildSiteTrendMergesSitesAndFillsBuckets(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	points := []AccountTrendPoint{
		{Bucket: "2026-01-01T00:00:00", AccountID: 1, BaseURL: "https://a.com/v1", Requests: 5, Errors: 1},
		{Bucket: "2026-01-01T02:00:00", AccountID: 2, BaseURL: "https://A.com", Requests: 3},
		{Bucket: "2026-01-01T01:00:00", AccountID: 3, BaseURL: "https://b.com", Requests: 4},
		{Bucket: "2026-01-01T01:00:00", AccountID: 4, Name: "oauth", Requests: 1},
	}
	trend := BuildSiteTrend(points, start, end, "hour", time.UTC, 1)
	if len(trend.Buckets) != 3 {
		t.Fatalf("want 3 buckets, got %v", trend.Buckets)
	}
	if len(trend.Series) != 2 || trend.Series[0].Label != "a.com" || trend.Series[0].AccountCount != 2 {
		t.Fatalf("unexpected series: %+v", trend.Series)
	}
	if got := trend.Series[0].Requests; got[0] != 5 || got[1] != 0 || got[2] != 3 {
		t.Fatalf("unexpected a.com requests: %v", got)
	}
	other := trend.Series[1]
	if other.Key != OperationsOtherSiteKey || other.TotalReq != 5 || other.AccountCount != 2 {
		t.Fatalf("unexpected other: %+v", other)
	}
	if trend.Total.TotalReq != 13 || trend.Total.TotalErrors != 1 {
		t.Fatalf("unexpected total: %+v", trend.Total)
	}
}

func TestBuildAccountHealth(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	old := now.Add(-30 * 24 * time.Hour)
	recent := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	accounts := []Account{
		{ID: 1, Platform: "openai", Status: StatusActive, Schedulable: true, CreatedAt: old, LastUsedAt: &recent},
		{ID: 2, Platform: "openai", Status: StatusActive, Schedulable: true, CreatedAt: old},
		{ID: 3, Platform: "anthropic", Status: StatusError, ErrorMessage: "401", CreatedAt: old},
		{ID: 4, Platform: "openai", Status: StatusActive, Schedulable: true, CreatedAt: old, RateLimitResetAt: &future},
		{ID: 5, Platform: "openai", Status: StatusActive, Schedulable: false, CreatedAt: old},
		{ID: 6, Platform: "openai", Status: StatusActive, Schedulable: true, CreatedAt: now.Add(-time.Hour)},
	}
	h := BuildAccountHealth(accounts, now)
	s := h.Summary
	if s.Total != 6 || s.Error != 1 || s.RateLimited != 1 || s.Paused != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
	if len(h.Idle) != 1 || h.Idle[0].ID != 2 {
		t.Fatalf("want only account 2 idle (6 is too new), got %+v", h.Idle)
	}
	if len(h.Issues) != 3 || h.Issues[0].Kind != HealthIssueError || h.Issues[1].Kind != HealthIssueRateLimited {
		t.Fatalf("unexpected issues order: %+v", h.Issues)
	}
	if h.Platforms[0].Platform != "openai" || h.Platforms[0].Total != 5 {
		t.Fatalf("unexpected platforms: %+v", h.Platforms)
	}
}
