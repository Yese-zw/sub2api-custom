package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAccountListRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts", handler.List)
	return router, adminSvc
}

func TestAccountHandlerListIncludesCreatedAt(t *testing.T) {
	router, adminSvc := setupAccountListRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=1&page_size=20&sort_by=created_at&sort_order=desc", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "created_at", adminSvc.lastListAccounts.sortBy)

	var payload struct {
		Data struct {
			Items []struct {
				ID        int64  `json:"id"`
				CreatedAt string `json:"created_at"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)

	createdAt := payload.Data.Items[0].CreatedAt
	require.NotEmpty(t, createdAt)
	require.True(t, strings.HasSuffix(createdAt, "Z"), "created_at should be serialized as UTC")
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	require.NoError(t, err)
	_, offset := parsed.Zone()
	require.Equal(t, 0, offset)
}

func TestAccountHandlerListDedupesByUpstreamBalanceURLPreferConfigured(t *testing.T) {
	router, adminSvc := setupAccountListRouter()
	adminSvc.accounts = []service.Account{
		{
			ID:     1,
			Name:   "plain",
			Type:   service.AccountTypeAPIKey,
			Status: service.StatusActive,
			Credentials: map[string]any{
				"base_url": "https://example.com/v1",
			},
			Schedulable: true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:     2,
			Name:   "configured",
			Type:   service.AccountTypeAPIKey,
			Status: service.StatusActive,
			Credentials: map[string]any{
				"base_url": "https://EXAMPLE.com",
			},
			Extra: map[string]any{
				service.UpstreamBalanceConfigExtraKey: map[string]any{
					"mode": service.UpstreamBalanceModeNewAPI,
				},
			},
			Schedulable: true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:     3,
			Name:   "other",
			Type:   service.AccountTypeAPIKey,
			Status: service.StatusActive,
			Credentials: map[string]any{
				"base_url": "https://other.example.com",
			},
			Schedulable: true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=1&page_size=20&type=apikey,upstream&dedupe_by_upstream_balance_url=true", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data struct {
			Items []struct {
				ID int64 `json:"id"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, int64(2), payload.Data.Total)
	require.Len(t, payload.Data.Items, 2)
	require.Equal(t, int64(2), payload.Data.Items[0].ID)
	require.Equal(t, int64(3), payload.Data.Items[1].ID)
}
