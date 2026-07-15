package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingsPUT_EmailConfig_RoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &settingHandlerRepoStub{values: map[string]string{
		service.SettingKeyRegistrationEnabled: "true",
		service.SettingKeyEmailVerifyEnabled:  "true",
		service.SettingKeyEmailProvider:       service.EmailProviderSMTP,
		service.SettingKeySMTPHost:            "smtp.old.example.com",
		service.SettingKeySMTPPort:            "587",
		service.SettingKeySMTPUsername:        "old-user",
		service.SettingKeySMTPPassword:        "old-pass",
		service.SettingKeySMTPFrom:            "old@example.com",
		service.SettingKeySMTPFromName:        "Old Sender",
		service.SettingKeySMTPUseTLS:          "true",
		service.SettingKeyResendAPIKey:        "",
		service.SettingKeyCloudflareAccountID: "",
		service.SettingKeyCloudflareEmailAPIToken: "",
	}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	payload := map[string]any{
		"registration_enabled": true,
		"email_verify_enabled": true,
		"email_provider": "resend",
		"smtp_host": "smtp.new.example.com",
		"smtp_port": 2525,
		"smtp_username": "new-user",
		"smtp_password": "new-pass",
		"smtp_from_email": "new@example.com",
		"smtp_from_name": "New Sender",
		"smtp_use_tls": false,
		"resend_api_key": "resend-secret",
		"cloudflare_email_account_id": "cf-acc",
		"cloudflare_email_api_token": "cf-token",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "resend", data["email_provider"])
	require.Equal(t, "smtp.new.example.com", data["smtp_host"])
	require.Equal(t, "new-user", data["smtp_username"])
	require.Equal(t, "new@example.com", data["smtp_from_email"])
	require.Equal(t, false, data["smtp_use_tls"])
	require.Equal(t, true, data["resend_api_key_configured"])
	require.Equal(t, "cf-acc", data["cloudflare_email_account_id"])
	require.Equal(t, true, data["cloudflare_email_api_token_configured"])
}
