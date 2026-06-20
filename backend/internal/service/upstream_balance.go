package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"math"
	"net/mail"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"golang.org/x/sync/errgroup"
)

const upstreamBalanceBodyLimit int64 = 1 << 20
const upstreamBalanceTimeout = 15 * time.Second
const newAPIQuotaPointsPerUSD = 500000.0
const upstreamBalanceRefreshPageSize = 1000
const upstreamBalanceRefreshMaxConcurrency = 10
const upstreamBalanceRefreshInterval = 10 * time.Minute
const upstreamBalanceRefreshFreshTTL = 5 * time.Minute

const (
	UpstreamBalanceConfigExtraKey        = "upstream_balance_config"
	UpstreamBalanceAccessKeyCredential  = "upstream_balance_access_key"
	UpstreamBalanceConfigNewAPIUserID   = "new_api_user_id"
	UpstreamBalanceConfigNotifyEnabled  = "low_balance_notify_enabled"
	UpstreamBalanceConfigNotifyThreshold = "low_balance_notify_threshold"
	UpstreamBalanceConfigNotifyEmails   = "low_balance_notify_emails"
	UpstreamBalanceConfigNotifyLastKey  = "low_balance_notify_last_key"
)

var ErrUpstreamBalanceMissingCredentials = errors.New("upstream balance missing credentials")

type UpstreamBalanceMode string

const (
	UpstreamBalanceModeSub2API UpstreamBalanceMode = "sub2api"
	UpstreamBalanceModeNewAPI  UpstreamBalanceMode = "new_api"
)

type UpstreamBalanceSnapshot struct {
	Mode             string     `json:"mode"`
	Balance          *float64   `json:"balance,omitempty"`
	Remaining        *float64   `json:"remaining,omitempty"`
	QuotaLimit       *float64   `json:"quota_limit,omitempty"`
	QuotaUsed        *float64   `json:"quota_used,omitempty"`
	QuotaRemaining   *float64   `json:"quota_remaining,omitempty"`
	QuotaUnlimited   *bool      `json:"quota_unlimited,omitempty"`
	Unit             string     `json:"unit,omitempty"`
	Status           string     `json:"status,omitempty"`
	PlanName         string     `json:"plan_name,omitempty"`
	TodayRequests    *int64     `json:"today_requests,omitempty"`
	TodayTotalTokens *int64     `json:"today_total_tokens,omitempty"`
	TodayCost        *float64   `json:"today_cost,omitempty"`
	TotalRequests    *int64     `json:"total_requests,omitempty"`
	TotalTotalTokens *int64     `json:"total_total_tokens,omitempty"`
	TotalCost        *float64   `json:"total_cost,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LatencyMs        int64      `json:"latency_ms"`
	Error            string     `json:"error,omitempty"`
}

type UpstreamBalanceService struct {
	cfg                  *config.Config
	balanceNotifyService *BalanceNotifyService
	adminService         UpstreamBalanceAccountLister
}

type UpstreamBalanceAccountLister interface {
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Account, int64, error)
	GetAccount(ctx context.Context, id int64) (*Account, error)
	UpdateAccountExtra(ctx context.Context, id int64, updates map[string]any) error
}

type UpstreamBalanceRefreshOptions struct {
	Platform                  string
	Status                    string
	Search                    string
	SkipRecentlyFresh         bool
	NotifyLowBalance          bool
	FreshTTL                  time.Duration
	PageSize                  int
	MaxConcurrency            int
}

type UpstreamBalanceRefreshResult struct {
	Items   []any
	Success int
	Skipped int
	Failed  int
	Errors  []map[string]any
}

func NewUpstreamBalanceService(cfg *config.Config) *UpstreamBalanceService {
	return &UpstreamBalanceService{cfg: cfg}
}

func (s *UpstreamBalanceService) SetBalanceNotifyService(balanceNotifyService *BalanceNotifyService) {
	if s != nil {
		s.balanceNotifyService = balanceNotifyService
	}
}

func (s *UpstreamBalanceService) SetAccountLister(adminService UpstreamBalanceAccountLister) {
	if s != nil {
		s.adminService = adminService
	}
}

func (s *UpstreamBalanceService) RefreshAccounts(ctx context.Context, opts UpstreamBalanceRefreshOptions, buildItem func(context.Context, *Account) any) (UpstreamBalanceRefreshResult, error) {
	var result UpstreamBalanceRefreshResult
	if s == nil {
		return result, fmt.Errorf("upstream balance service unavailable")
	}
	admin := s.adminService
	if admin == nil {
		return result, fmt.Errorf("upstream balance account lister is required")
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = upstreamBalanceRefreshPageSize
	}
	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = upstreamBalanceRefreshMaxConcurrency
	}
	freshTTL := opts.FreshTTL
	if freshTTL <= 0 {
		freshTTL = upstreamBalanceRefreshFreshTTL
	}
	search := strings.TrimSpace(opts.Search)
	if len(search) > 100 {
		search = search[:100]
	}

	var mu sync.Mutex
	for page := 1; ; page++ {
		accounts, total, err := admin.ListAccounts(ctx, page, pageSize, opts.Platform, "apikey,upstream", opts.Status, search, 0, "", "schedulable", "desc")
		if err != nil {
			return result, err
		}
		if len(accounts) == 0 {
			break
		}

		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(maxConcurrency)
		for i := range accounts {
			account := accounts[i]
			if opts.SkipRecentlyFresh && isUpstreamBalanceFresh(&account, freshTTL, time.Now()) {
				mu.Lock()
				result.Skipped++
				mu.Unlock()
				continue
			}
			g.Go(func() error {
				item, err := s.refreshOneAccount(gctx, admin, &account, opts.NotifyLowBalance, buildItem)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, map[string]any{
						"account_id": account.ID,
						"error":      err.Error(),
					})
					return nil
				}
				result.Success++
				if item != nil {
					result.Items = append(result.Items, item)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return result, err
		}

		if int64(page*pageSize) >= total {
			break
		}
	}
	return result, nil
}

func (s *UpstreamBalanceService) refreshOneAccount(ctx context.Context, admin UpstreamBalanceAccountLister, account *Account, notifyLowBalance bool, buildItem func(context.Context, *Account) any) (any, error) {
	snapshot, err := s.Query(ctx, account)
	if err != nil {
		slog.Warn("upstream_balance_batch_refresh_failed", "account_id", account.ID, "error", err)
		return nil, err
	}
	if err := admin.UpdateAccountExtra(ctx, account.ID, map[string]any{
		"upstream_balance": snapshot,
	}); err != nil {
		slog.Warn("upstream_balance_batch_save_failed", "account_id", account.ID, "error", err)
		return nil, err
	}
	if notifyLowBalance {
		if updates := s.MaybeNotifyLowBalance(ctx, account, snapshot); len(updates) > 0 {
			if err := admin.UpdateAccountExtra(ctx, account.ID, updates); err != nil {
				slog.Warn("upstream_balance_low_notify_state_update_failed", "account_id", account.ID, "error", err)
			}
		}
	}
	if buildItem == nil {
		return nil, nil
	}
	updated, err := admin.GetAccount(ctx, account.ID)
	if err != nil {
		slog.Warn("upstream_balance_batch_reload_failed", "account_id", account.ID, "error", err)
		return nil, err
	}
	return buildItem(ctx, updated), nil
}

func isUpstreamBalanceFresh(account *Account, ttl time.Duration, now time.Time) bool {
	if ttl <= 0 || account == nil || account.Extra == nil {
		return false
	}
	raw, ok := account.Extra["upstream_balance"]
	if !ok {
		return false
	}
	updatedAt := upstreamBalanceSnapshotUpdatedAt(raw)
	return !updatedAt.IsZero() && now.Sub(updatedAt) < ttl
}

func upstreamBalanceSnapshotUpdatedAt(raw any) time.Time {
	switch v := raw.(type) {
	case UpstreamBalanceSnapshot:
		return v.UpdatedAt
	case *UpstreamBalanceSnapshot:
		if v != nil {
			return v.UpdatedAt
		}
	case map[string]any:
		return parseUpstreamBalanceUpdatedAt(v["updated_at"])
	case map[string]string:
		return parseUpstreamBalanceUpdatedAt(v["updated_at"])
	}
	return time.Time{}
}

func parseUpstreamBalanceUpdatedAt(raw any) time.Time {
	switch v := raw.(type) {
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v)); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(v)); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (s *UpstreamBalanceService) Query(ctx context.Context, account *Account) (*UpstreamBalanceSnapshot, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if account.Type != AccountTypeAPIKey && account.Type != AccountTypeUpstream {
		return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_UNSUPPORTED_ACCOUNT", "only API key upstream accounts support upstream balance query")
	}

	baseURL := upstreamBalanceBaseURL(account)
	if baseURL == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_BASE_URL", "account base URL is required")
	}
	normalizedBaseURL, err := s.validateBaseURL(baseURL)
	if err != nil {
		return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_INVALID_BASE_URL", "invalid account base URL")
	}

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              upstreamBalanceProxyURL(account),
		Timeout:               upstreamBalanceTimeout,
		ResponseHeaderTimeout: upstreamBalanceTimeout,
		ValidateResolvedIP:    true,
		AllowPrivateHosts:     s.allowPrivateHosts(),
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream balance client: %w", err)
	}

	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	accountAccessKey := strings.TrimSpace(account.GetCredential(UpstreamBalanceAccessKeyCredential))
	newAPIUserID := upstreamBalanceConfigString(account, UpstreamBalanceConfigNewAPIUserID)
	requestedMode := upstreamBalanceConfigMode(account)
	if requestedMode == "" {
		requestedMode = normalizeUpstreamBalanceMode(account.GetCredential("usage_integration_type"))
	}
	if requestedMode == "" {
		requestedMode = normalizeUpstreamBalanceMode(account.GetExtraString("usage_integration_type"))
	}
	if requestedMode == "" {
		requestedMode = detectUpstreamBalanceMode(normalizedBaseURL)
	}

	switch requestedMode {
	case UpstreamBalanceModeNewAPI:
		if accountAccessKey == "" {
			return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_ACCESS_KEY", "new-api account access key is required").WithCause(ErrUpstreamBalanceMissingCredentials)
		}
		if newAPIUserID == "" {
			return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_NEW_API_USER_ID", "new-api user ID is required").WithCause(ErrUpstreamBalanceMissingCredentials)
		}
		return s.queryNewAPIUserSelf(ctx, client, normalizedBaseURL, accountAccessKey, newAPIUserID)
	case UpstreamBalanceModeSub2API:
		if apiKey == "" {
			return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_API_KEY", "account API key is required").WithCause(ErrUpstreamBalanceMissingCredentials)
		}
		return s.querySub2API(ctx, client, normalizedBaseURL, apiKey)
	default:
		var attempted bool
		var newAPIErr error = ErrUpstreamBalanceMissingCredentials
		if accountAccessKey != "" && newAPIUserID != "" {
			attempted = true
			newAPI, err := s.queryNewAPIUserSelf(ctx, client, normalizedBaseURL, accountAccessKey, newAPIUserID)
			if err == nil {
				return newAPI, nil
			}
			newAPIErr = err
		} else if accountAccessKey != "" {
			newAPIErr = infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_NEW_API_USER_ID", "new-api user ID is required").WithCause(ErrUpstreamBalanceMissingCredentials)
		}
		var sub2apiErr error = ErrUpstreamBalanceMissingCredentials
		if apiKey != "" {
			attempted = true
			sub2api, err := s.querySub2API(ctx, client, normalizedBaseURL, apiKey)
			if err == nil {
				return sub2api, nil
			}
			sub2apiErr = err
		}
		if !attempted {
			return nil, infraerrors.BadRequest("UPSTREAM_BALANCE_MISSING_CREDENTIALS", "account API key or new-api account access key and user ID is required").WithCause(ErrUpstreamBalanceMissingCredentials)
		}
		return nil, fmt.Errorf("detect upstream balance endpoint failed: new_api: %v; sub2api: %v", newAPIErr, sub2apiErr)
	}
}

func (s *UpstreamBalanceService) validateBaseURL(raw string) (string, error) {
	if s == nil || s.cfg == nil {
		return urlvalidator.ValidateURLFormat(raw, false)
	}
	if !s.cfg.Security.URLAllowlist.Enabled {
		return urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	return urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
}

func (s *UpstreamBalanceService) allowPrivateHosts() bool {
	return s != nil && s.cfg != nil && s.cfg.Security.URLAllowlist.AllowPrivateHosts
}

func (s *UpstreamBalanceService) querySub2API(ctx context.Context, client *http.Client, baseURL, apiKey string) (*UpstreamBalanceSnapshot, error) {
	usageURLs, err := sub2APIUsageURLs(baseURL)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	var body []byte
	var lastErr error
	for _, usageURL := range usageURLs {
		body, lastErr = doUpstreamBalanceGET(ctx, client, usageURL, apiKey)
		if lastErr == nil && !looksLikeHTML(body) {
			break
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("sub2api usage response is HTML")
		}
		body = nil
	}
	latencyMs := time.Since(started).Milliseconds()
	if lastErr != nil {
		return nil, lastErr
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse sub2api usage response: %w", err)
	}

	snapshot := &UpstreamBalanceSnapshot{
		Mode:             string(UpstreamBalanceModeSub2API),
		Balance:          jsonFloat(raw, "balance"),
		Remaining:        jsonFloat(raw, "remaining"),
		QuotaLimit:       nestedJSONFloat(raw, "quota", "limit"),
		QuotaUsed:        nestedJSONFloat(raw, "quota", "used"),
		QuotaRemaining:   nestedJSONFloat(raw, "quota", "remaining"),
		QuotaUnlimited:   nestedJSONBool(raw, "quota", "unlimited"),
		Unit:             firstJSONString(raw, "unit"),
		Status:           firstJSONString(raw, "status"),
		PlanName:         firstJSONString(raw, "planName", "plan_name"),
		TodayRequests:    nestedJSONInt(raw, "usage", "today", "requests"),
		TodayTotalTokens: nestedJSONInt(raw, "usage", "today", "total_tokens"),
		TodayCost:        nestedJSONFloat(raw, "usage", "today", "cost"),
		TotalRequests:    nestedJSONInt(raw, "usage", "total", "requests"),
		TotalTotalTokens: nestedJSONInt(raw, "usage", "total", "total_tokens"),
		TotalCost:        nestedJSONFloat(raw, "usage", "total", "cost"),
		UpdatedAt:        time.Now().UTC(),
		LatencyMs:        latencyMs,
	}
	if snapshot.Unit == "" {
		snapshot.Unit = "USD"
	}
	return snapshot, nil
}

func (s *UpstreamBalanceService) queryNewAPIUserSelf(ctx context.Context, client *http.Client, baseURL, accountAccessKey, userID string) (*UpstreamBalanceSnapshot, error) {
	selfURL, err := newAPIUserSelfURL(baseURL)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	body, err := doUpstreamBalanceGETWithHeaders(ctx, client, selfURL, accountAccessKey, map[string]string{
		"Content-Type": "application/json",
		"New-Api-User": userID,
	})
	latencyMs := time.Since(started).Milliseconds()
	if err != nil {
		return nil, err
	}

	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse new-api user response: %w", err)
	}
	if success, ok := root["success"].(bool); ok && !success {
		message := firstJSONString(root, "message", "error")
		if message == "" {
			message = "new-api user balance query failed"
		}
		return nil, fmt.Errorf("%s", message)
	}

	data, ok := root["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("new-api user response missing data")
	}

	quotaPoints := firstJSONFloat(data, "quota")
	usedQuotaPoints := firstJSONFloat(data, "used_quota")
	var balanceUSD *float64
	if quotaPoints != nil {
		v := *quotaPoints / newAPIQuotaPointsPerUSD
		balanceUSD = &v
	}
	var usedUSD *float64
	if usedQuotaPoints != nil {
		v := *usedQuotaPoints / newAPIQuotaPointsPerUSD
		usedUSD = &v
	}

	return &UpstreamBalanceSnapshot{
		Mode:           string(UpstreamBalanceModeNewAPI),
		Balance:        balanceUSD,
		Remaining:      balanceUSD,
		QuotaUsed:      usedUSD,
		QuotaRemaining: balanceUSD,
		Unit:           "USD",
		Status:         firstJSONString(data, "status"),
		PlanName:       firstJSONString(data, "username"),
		UpdatedAt:      time.Now().UTC(),
		LatencyMs:      latencyMs,
	}, nil
}

func doUpstreamBalanceGET(ctx context.Context, client *http.Client, targetURL, apiKey string) ([]byte, error) {
	return doUpstreamBalanceGETWithHeaders(ctx, client, targetURL, apiKey, nil)
}

func doUpstreamBalanceGETWithHeaders(ctx context.Context, client *http.Client, targetURL, apiKey string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamBalanceBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > upstreamBalanceBodyLimit {
		return nil, fmt.Errorf("upstream balance response is too large")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		previewLen := len(body)
		if previewLen > 300 {
			previewLen = 300
		}
		return nil, fmt.Errorf("upstream balance HTTP %d: %s", resp.StatusCode, string(body[:previewLen]))
	}
	return body, nil
}

func upstreamBalanceBaseURL(account *Account) string {
	if account == nil {
		return ""
	}
	return strings.TrimSpace(account.GetCredential("base_url"))
}

func UpstreamBalanceRequestDedupKey(account *Account) string {
	if account == nil {
		return ""
	}
	baseURL := strings.TrimSpace(upstreamBalanceBaseURL(account))
	if baseURL == "" {
		return ""
	}
	mode := upstreamBalanceConfigMode(account)
	if mode == "" {
		mode = normalizeUpstreamBalanceMode(account.GetCredential("usage_integration_type"))
	}
	if mode == "" {
		mode = normalizeUpstreamBalanceMode(account.GetExtraString("usage_integration_type"))
	}
	if mode == "" {
		mode = detectUpstreamBalanceMode(baseURL)
	}
	accountAccessKey := strings.TrimSpace(account.GetCredential(UpstreamBalanceAccessKeyCredential))
	newAPIUserID := upstreamBalanceConfigString(account, UpstreamBalanceConfigNewAPIUserID)
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))

	var targetURL string
	var err error
	switch mode {
	case UpstreamBalanceModeNewAPI:
		targetURL, err = newAPIUserSelfURL(baseURL)
	case UpstreamBalanceModeSub2API:
		urls, urlsErr := sub2APIUsageURLs(baseURL)
		if urlsErr == nil && len(urls) > 0 {
			targetURL = urls[0]
		}
		err = urlsErr
	default:
		if accountAccessKey != "" && newAPIUserID != "" {
			targetURL, err = newAPIUserSelfURL(baseURL)
		} else if apiKey != "" {
			urls, urlsErr := sub2APIUsageURLs(baseURL)
			if urlsErr == nil && len(urls) > 0 {
				targetURL = urls[0]
			}
			err = urlsErr
		} else {
			targetURL, err = newAPIUserSelfURL(baseURL)
		}
	}
	if err != nil || strings.TrimSpace(targetURL) == "" {
		return ""
	}
	return normalizeUpstreamBalanceRequestURL(targetURL)
}

func HasUpstreamBalanceConfig(account *Account) bool {
	if account == nil || account.Extra == nil {
		return false
	}
	raw, ok := account.Extra[UpstreamBalanceConfigExtraKey]
	if !ok || raw == nil {
		return false
	}
	if config, ok := raw.(map[string]any); ok {
		return len(config) > 0
	}
	return true
}

func normalizeUpstreamBalanceRequestURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func upstreamBalanceProxyURL(account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func normalizeUpstreamBalanceMode(value string) UpstreamBalanceMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(UpstreamBalanceModeSub2API), "sub2-api":
		return UpstreamBalanceModeSub2API
	case string(UpstreamBalanceModeNewAPI), "new-api":
		return UpstreamBalanceModeNewAPI
	default:
		return ""
	}
}

func NormalizeUpstreamBalanceModeForConfig(value string) UpstreamBalanceMode {
	return normalizeUpstreamBalanceMode(value)
}

func UpstreamBalanceConfigFromAccount(account *Account) map[string]any {
	return upstreamBalanceConfigMap(account)
}

func upstreamBalanceConfigMode(account *Account) UpstreamBalanceMode {
	return normalizeUpstreamBalanceMode(upstreamBalanceConfigString(account, "mode"))
}

func upstreamBalanceConfigString(account *Account, key string) string {
	if account == nil || account.Extra == nil {
		return ""
	}
	raw, ok := account.Extra[UpstreamBalanceConfigExtraKey]
	if !ok {
		return ""
	}
	config, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}

func detectUpstreamBalanceMode(baseURL string) UpstreamBalanceMode {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, "new-api") {
		return UpstreamBalanceModeNewAPI
	}
	if strings.Contains(host, "sub2api") || strings.Contains(host, "apikey.fun") {
		return UpstreamBalanceModeSub2API
	}
	return ""
}

func appendPath(baseURL, endpoint string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(strings.TrimRight(baseURL, "/")))
	if err != nil {
		return "", err
	}
	basePath := strings.TrimRight(u.Path, "/")
	nextPath := strings.Trim(endpoint, "/")
	if basePath == "" {
		u.Path = "/" + nextPath
	} else {
		u.Path = basePath + "/" + nextPath
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func sub2APIUsageURLs(baseURL string) ([]string, error) {
	usageURL, err := appendPath(baseURL, "usage")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(strings.TrimSpace(strings.TrimRight(baseURL, "/")))
	if err != nil {
		return nil, err
	}
	basePath := strings.TrimRight(u.Path, "/")
	if basePath == "/v1" {
		return []string{usageURL}, nil
	}
	v1UsageURL, err := appendPath(baseURL, "v1/usage")
	if err != nil {
		return nil, err
	}
	if v1UsageURL == usageURL {
		return []string{usageURL}, nil
	}
	return []string{usageURL, v1UsageURL}, nil
}

func looksLikeHTML(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	return strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html") ||
		strings.HasPrefix(strings.ToLower(trimmed), "<html")
}

func newAPIUserSelfURL(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(strings.TrimRight(baseURL, "/")))
	if err != nil {
		return "", err
	}
	basePath := strings.TrimRight(u.Path, "/")
	if basePath == "/v1" {
		basePath = ""
	} else if strings.HasSuffix(basePath, "/v1") {
		basePath = strings.TrimSuffix(basePath, "/v1")
	}
	nextPath := "api/user/self"
	if basePath == "" {
		u.Path = "/" + nextPath
	} else {
		u.Path = basePath + "/" + nextPath
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func firstJSONString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func firstJSONFloat(raw map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if value := jsonFloat(raw, key); value != nil {
			return value
		}
	}
	return nil
}

func firstJSONBool(raw map[string]any, keys ...string) *bool {
	for _, key := range keys {
		if value := jsonBool(raw, key); value != nil {
			return value
		}
	}
	return nil
}

func jsonBool(raw map[string]any, key string) *bool {
	if raw == nil {
		return nil
	}
	switch v := raw[key].(type) {
	case bool:
		return &v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes":
			b := true
			return &b
		case "false", "0", "no":
			b := false
			return &b
		}
	}
	return nil
}

func jsonFloat(raw map[string]any, key string) *float64 {
	if raw == nil {
		return nil
	}
	switch v := raw[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return &f
		}
	case string:
		if f, err := json.Number(strings.TrimSpace(v)).Float64(); err == nil {
			return &f
		}
	}
	return nil
}

func nestedJSONFloat(raw map[string]any, path ...string) *float64 {
	if len(path) == 0 {
		return nil
	}
	current := raw
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return jsonFloat(current, path[len(path)-1])
}

func nestedJSONInt(raw map[string]any, path ...string) *int64 {
	if len(path) == 0 {
		return nil
	}
	current := raw
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return jsonInt(current, path[len(path)-1])
}

func jsonInt(raw map[string]any, key string) *int64 {
	if raw == nil {
		return nil
	}
	switch v := raw[key].(type) {
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	case int64:
		return &v
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return &i
		}
	case string:
		if i, err := json.Number(strings.TrimSpace(v)).Int64(); err == nil {
			return &i
		}
	}
	return nil
}

func nestedJSONBool(raw map[string]any, path ...string) *bool {
	if len(path) == 0 {
		return nil
	}
	current := raw
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	if v, ok := current[path[len(path)-1]].(bool); ok {
		return &v
	}
	return nil
}

func (s *UpstreamBalanceService) MaybeNotifyLowBalance(ctx context.Context, account *Account, snapshot *UpstreamBalanceSnapshot) map[string]any {
	if s == nil || s.balanceNotifyService == nil || account == nil || snapshot == nil {
		return nil
	}
	config := upstreamBalanceConfigMap(account)
	if !upstreamBalanceConfigBool(config, UpstreamBalanceConfigNotifyEnabled) {
		return nil
	}
	threshold := upstreamBalanceConfigFloat(config, UpstreamBalanceConfigNotifyThreshold)
	if threshold <= 0 || math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return nil
	}
	recipients := normalizeUpstreamBalanceNotifyEmails(upstreamBalanceConfigStringSlice(config, UpstreamBalanceConfigNotifyEmails))
	if len(recipients) == 0 {
		return nil
	}
	balance, ok := upstreamBalanceSnapshotValue(snapshot)
	if snapshot.QuotaUnlimited != nil && *snapshot.QuotaUnlimited {
		return clearUpstreamBalanceLowNotifyState(config)
	}
	if !ok {
		return nil
	}
	ratio := upstreamBalanceConfigFloat(config, "balance_ratio")
	if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		ratio = 1
	}
	adjustedBalance := balance * ratio
	if adjustedBalance > threshold {
		return clearUpstreamBalanceLowNotifyState(config)
	}
	notifyKey := upstreamBalanceLowNotifyKey(threshold, ratio, recipients)
	if notifyKey != "" && notifyKey == upstreamBalanceConfigStringFromMap(config, UpstreamBalanceConfigNotifyLastKey) {
		return nil
	}

	s.balanceNotifyService.SendUpstreamBalanceLowAlert(ctx, recipients, account.ID, account.Name, snapshot.Mode, balance, adjustedBalance, threshold, snapshot.Unit)
	nextConfig := copyStringAnyMap(config)
	nextConfig[UpstreamBalanceConfigNotifyLastKey] = notifyKey
	return map[string]any{UpstreamBalanceConfigExtraKey: nextConfig}
}

func clearUpstreamBalanceLowNotifyState(config map[string]any) map[string]any {
	if upstreamBalanceConfigStringFromMap(config, UpstreamBalanceConfigNotifyLastKey) == "" {
		return nil
	}
	nextConfig := copyStringAnyMap(config)
	delete(nextConfig, UpstreamBalanceConfigNotifyLastKey)
	return map[string]any{UpstreamBalanceConfigExtraKey: nextConfig}
}

func upstreamBalanceConfigMap(account *Account) map[string]any {
	if account == nil || account.Extra == nil {
		return nil
	}
	raw, _ := account.Extra[UpstreamBalanceConfigExtraKey].(map[string]any)
	return raw
}

func copyStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func upstreamBalanceConfigBool(config map[string]any, key string) bool {
	if config == nil {
		return false
	}
	switch v := config[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func upstreamBalanceConfigFloat(config map[string]any, key string) float64 {
	if config == nil {
		return 0
	}
	switch v := config[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func upstreamBalanceConfigStringFromMap(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}

func upstreamBalanceConfigStringSlice(config map[string]any, key string) []string {
	if config == nil {
		return nil
	}
	switch v := config[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	case string:
		return splitUpstreamBalanceNotifyEmails(v)
	default:
		return nil
	}
}

func splitUpstreamBalanceNotifyEmails(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	return fields
}

func normalizeUpstreamBalanceNotifyEmails(emails []string) []string {
	seen := make(map[string]struct{}, len(emails))
	out := make([]string, 0, len(emails))
	for _, raw := range emails {
		email := strings.TrimSpace(raw)
		if email == "" || len(email) > 254 {
			continue
		}
		if addr, err := mail.ParseAddress(email); err == nil && strings.EqualFold(strings.TrimSpace(addr.Address), email) {
			key := strings.ToLower(email)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, email)
		}
	}
	return out
}

func upstreamBalanceSnapshotValue(snapshot *UpstreamBalanceSnapshot) (float64, bool) {
	if snapshot == nil {
		return 0, false
	}
	for _, value := range []*float64{snapshot.QuotaRemaining, snapshot.Remaining, snapshot.Balance} {
		if value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0) {
			return *value, true
		}
	}
	return 0, false
}

func UpstreamBalanceLowNotifyKey(threshold, ratio float64, recipients []string) string {
	return upstreamBalanceLowNotifyKey(threshold, ratio, recipients)
}

func upstreamBalanceLowNotifyKey(threshold, ratio float64, recipients []string) string {
	emails := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		email := strings.ToLower(strings.TrimSpace(recipient))
		if email != "" {
			emails = append(emails, email)
		}
	}
	sort.Strings(emails)
	return fmt.Sprintf("threshold:%.4f;ratio:%.6f;emails:%s", threshold, ratio, strings.Join(emails, ","))
}

func (s *BalanceNotifyService) SendUpstreamBalanceLowAlert(ctx context.Context, recipients []string, accountID int64, accountName, mode string, upstreamBalance, adjustedBalance, threshold float64, unit string) {
	if s == nil || s.emailService == nil || len(recipients) == 0 {
		return
	}
	recipients = normalizeUpstreamBalanceNotifyEmails(recipients)
	if len(recipients) == 0 {
		return
	}
	siteName := s.getSiteName(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in upstream balance low notification", "recover", r)
			}
		}()
		s.sendUpstreamBalanceLowAlertEmails(recipients, accountID, accountName, mode, upstreamBalance, adjustedBalance, threshold, unit, siteName)
	}()
}

func (s *BalanceNotifyService) sendUpstreamBalanceLowAlertEmails(recipients []string, accountID int64, accountName, mode string, upstreamBalance, adjustedBalance, threshold float64, unit, siteName string) {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = "USD"
	}
	if s.notificationEmailService != nil {
		fallbackRecipients := make([]string, 0, len(recipients))
		for _, to := range recipients {
			ctx, cancel := context.WithTimeout(context.Background(), emailSendTimeout)
			err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
				Event:          NotificationEmailEventUpstreamBalanceLow,
				RecipientEmail: to,
				RecipientName:  emailRecipientName(to),
				SourceType:     "upstream_balance_low",
				SourceID:       strconv.FormatInt(accountID, 10),
				ReminderKey:    time.Now().UTC().Format("2006-01-02"),
				Variables: map[string]string{
					"account_id":        strconv.FormatInt(accountID, 10),
					"account_name":      accountName,
					"mode":              mode,
					"upstream_balance":  formatUpstreamBalanceAmount(upstreamBalance, unit),
					"adjusted_balance":  formatUpstreamBalanceAmount(adjustedBalance, unit),
					"balance_threshold": formatUpstreamBalanceAmount(threshold, unit),
				},
			})
			cancel()
			if err != nil {
				if shouldFallbackNotificationEmail(err) {
					slog.Warn("template upstream balance low alert failed; falling back to built-in body", "to", to, "account_id", accountID, "err", err.Error())
					fallbackRecipients = append(fallbackRecipients, to)
				} else {
					slog.Warn("template upstream balance low alert delivery failed; not sending fallback to avoid duplicates", "to", to, "account_id", accountID, "err", err.Error())
				}
			}
		}
		if len(fallbackRecipients) == 0 {
			return
		}
		recipients = fallbackRecipients
	}
	subject := fmt.Sprintf("[%s] 上游余额不足提醒 / Upstream Balance Low - %s", sanitizeEmailHeader(siteName), sanitizeEmailHeader(accountName))
	body := buildUpstreamBalanceLowEmailBody(
		html.EscapeString(siteName),
		accountID,
		html.EscapeString(accountName),
		html.EscapeString(mode),
		html.EscapeString(formatUpstreamBalanceAmount(upstreamBalance, unit)),
		html.EscapeString(formatUpstreamBalanceAmount(adjustedBalance, unit)),
		html.EscapeString(formatUpstreamBalanceAmount(threshold, unit)),
	)
	s.sendEmails(recipients, subject, body, "account_id", accountID, "account_name", accountName)
}

func formatUpstreamBalanceAmount(value float64, unit string) string {
	unit = strings.TrimSpace(unit)
	if strings.EqualFold(unit, "USD") || unit == "" {
		sign := ""
		if value < 0 {
			sign = "-"
			value = -value
		}
		return fmt.Sprintf("%s%.2f$", sign, value)
	}
	return fmt.Sprintf("%.2f %s", value, strings.ToUpper(unit))
}

func buildUpstreamBalanceLowEmailBody(siteName string, accountID int64, accountName, mode, upstreamBalance, adjustedBalance, threshold string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #ef4444 0%%, #b91c1c 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 36px 30px; }
        .metric { display: flex; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid #eee; }
        .metric-label { color: #666; }
        .metric-value { font-weight: bold; color: #333; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header"><h1>%s</h1></div>
        <div class="content">
            <p style="font-size: 18px; color: #333; text-align: center;">上游余额不足提醒 / Upstream Balance Low</p>
            <div class="metric"><span class="metric-label">账号 ID / Account ID</span><span class="metric-value">#%d</span></div>
            <div class="metric"><span class="metric-label">账号 / Account</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">模式 / Mode</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">上游余额 / Upstream Balance</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">折算余额 / Adjusted Balance</span><span class="metric-value">%s</span></div>
            <div class="metric"><span class="metric-label">提醒阈值 / Alert Threshold</span><span class="metric-value">%s</span></div>
            <p style="color:#666;line-height:1.6;margin-top:20px;text-align:center;">该账号折算余额已低于提醒阈值，请及时处理。</p>
        </div>
        <div class="footer"><p>此邮件由系统自动发送，请勿回复。</p></div>
    </div>
</body>
</html>`, siteName, accountID, accountName, mode, upstreamBalance, adjustedBalance, threshold)
}
