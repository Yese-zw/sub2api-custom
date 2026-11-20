package handler

import (
	"bytes"
	"compress/gzip"
	"html"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService           *service.SettingService
	notificationEmailService *service.NotificationEmailService
	version                  string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
}

// SetNotificationEmailService attaches the public notification email service without
// changing the constructor signature used by existing tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// GetPublicSettings 获取公开设置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PublicSettings{
		RegistrationEnabled:                 settings.RegistrationEnabled,
		EmailVerifyEnabled:                  settings.EmailVerifyEnabled,
		ForceEmailOnThirdPartySignup:        settings.ForceEmailOnThirdPartySignup,
		RegistrationEmailSuffixWhitelist:    settings.RegistrationEmailSuffixWhitelist,
		RegistrationEmailDomainQuotaEnabled: settings.RegistrationEmailDomainQuotaEnabled,
		PromoCodeEnabled:                    settings.PromoCodeEnabled,
		PasswordResetEnabled:                settings.PasswordResetEnabled,
		InvitationCodeEnabled:               settings.InvitationCodeEnabled,
		TotpEnabled:                         settings.TotpEnabled,
		PasskeyEnabled:                      settings.PasskeyEnabled,
		LoginAgreementEnabled:               settings.LoginAgreementEnabled,
		LoginAgreementMode:                  settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:             settings.LoginAgreementUpdatedAt,
		LoginAgreementRevision:              settings.LoginAgreementRevision,
		LoginAgreementDocuments:             publicLoginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		TurnstileEnabled:                    settings.TurnstileEnabled,
		TurnstileSiteKey:                    settings.TurnstileSiteKey,
		TencentCaptchaEnabled:               settings.TencentCaptchaEnabled,
		TencentCaptchaAppID:                 settings.TencentCaptchaAppID,
		TencentCaptchaRegion:                settings.TencentCaptchaRegion,
		AliyunCaptchaEnabled:                settings.AliyunCaptchaEnabled,
		AliyunCaptchaSceneID:                settings.AliyunCaptchaSceneID,
		AliyunCaptchaPrefix:                 settings.AliyunCaptchaPrefix,
		AliyunCaptchaRegion:                 settings.AliyunCaptchaRegion,
		SiteName:                            settings.SiteName,
		SiteLogo:                            settings.SiteLogo,
		CommunityGroupImage:                 settings.CommunityGroupImage,
		AppName:                             settings.AppName,
		AppDefaultGroupID:                   settings.AppDefaultGroupID,
		UITheme:                             settings.UITheme,
		SiteSubtitle:                        settings.SiteSubtitle,
		APIBaseURL:                          settings.APIBaseURL,
		ContactInfo:                         settings.ContactInfo,
		DocURL:                              settings.DocURL,
		HomeContent:                         settings.HomeContent,
		HomeContentURL:                      settings.HomeContentURL,
		CompactHomeEnabled:                  settings.CompactHomeEnabled,
		HideCcsImportButton:                 settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:         settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:             settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                settings.TableDefaultPageSize,
		TablePageSizeOptions:                settings.TablePageSizeOptions,
		CustomMenuItems:                     dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                     dto.ParseCustomEndpoints(settings.CustomEndpoints),
		DingTalkOAuthEnabled:                settings.DingTalkOAuthEnabled,
		LinuxDoOAuthEnabled:                 settings.LinuxDoOAuthEnabled,
		WeChatOAuthEnabled:                  settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:              settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:                settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:            settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                    settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:               settings.OIDCOAuthProviderName,
		GitHubOAuthEnabled:                  settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:                  settings.GoogleOAuthEnabled,
		BackendModeEnabled:                  settings.BackendModeEnabled,
		MCPEnabled:                          settings.MCPEnabled,
		PaymentEnabled:                      settings.PaymentEnabled,
		PaymentBalanceDisabled:              settings.PaymentBalanceDisabled,
		Version:                             h.version,
		ServerTimezone:                      timezone.Name(),
		ServerUTCOffset:                     timezone.UTCOffset(),
		BalanceLowNotifyEnabled:             settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:           settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:           settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:         settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorMode:                   settings.ChannelMonitorMode,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		ChannelMonitorHideThroughput:         settings.ChannelMonitorHideThroughput,
		ChannelMonitorShowQuota:              settings.ChannelMonitorShowQuota,
		ChannelMonitorHideUserRanking:        settings.ChannelMonitorHideUserRanking,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,
		SubscriptionEnabled:      settings.SubscriptionEnabled,

		ModelPlazaEnabled:       settings.ModelPlazaEnabled,
		ModelPlazaRequireAuth:   settings.ModelPlazaRequireAuth,
		PluginManagementEnabled: settings.PluginManagementEnabled,

		AffiliateEnabled: settings.AffiliateEnabled,

		RiskControlEnabled: settings.RiskControlEnabled,

		AllowUserViewErrorRequests: settings.AllowUserViewErrorRequests,
	})
}

// UnsubscribeNotificationEmail handles optional notification email opt-outs.
// GET /api/v1/settings/email-unsubscribe?token=...
func (h *SettingHandler) UnsubscribeNotificationEmail(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}
	result, err := h.notificationEmailService.Unsubscribe(c.Request.Context(), token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	body := "<!doctype html><html><head><meta charset=\"utf-8\"><title>Unsubscribed</title></head><body style=\"font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;padding:32px;\"><h1>Unsubscribed</h1><p>You have unsubscribed <strong>" + html.EscapeString(result.Email) + "</strong> from <strong>" + html.EscapeString(result.Event) + "</strong> emails.</p></body></html>"
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func publicLoginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}

// GetHomeContent serves a large custom homepage without putting it in the
// public settings payload used by every page.
// GET /api/v1/settings/home
func (h *SettingHandler) GetHomeContent(c *gin.Context) {
	body, etag, ok, err := h.settingService.PrepareHomeDocument(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	writePublicBody(c, body, "text/html; charset=utf-8", "public, max-age=60", etag, true)
}

// GetHomeAsset serves an image extracted from the custom homepage.
// GET /api/v1/settings/home-asset/:id
func (h *SettingHandler) GetHomeAsset(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	body, contentType, ok, err := h.settingService.HomeAsset(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	writePublicBody(c, body, contentType, "public, max-age=31536000, immutable", `"`+id+`"`, false)
}

func writePublicBody(c *gin.Context, body []byte, contentType, cacheControl, etag string, allowGzip bool) {
	c.Header("Cache-Control", cacheControl)
	if etag != "" {
		c.Header("ETag", etag)
	}
	c.Header("X-Content-Type-Options", "nosniff")
	if etag != "" && c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	if allowGzip && len(body) >= 256 && acceptsGzipHeader(c.GetHeader("Accept-Encoding")) {
		var buf bytes.Buffer
		zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
		if err == nil {
			_, writeErr := zw.Write(body)
			closeErr := zw.Close()
			if writeErr == nil && closeErr == nil && buf.Len() < len(body) {
				c.Header("Content-Encoding", "gzip")
				c.Header("Vary", "Accept-Encoding")
				c.Data(http.StatusOK, contentType, buf.Bytes())
				return
			}
		}
	}
	c.Data(http.StatusOK, contentType, body)
}

func acceptsGzipHeader(value string) bool {
	for _, part := range strings.Split(value, ",") {
		token := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if token == "gzip" {
			return true
		}
	}
	return false
}

// GetAppUpdateManifest 返回桌面 App 的 Tauri 更新清单（无需认证）
// GET /api/v1/app/update/:target
//
// 未发布或该平台无安装包时返回 204，Tauri updater 视为"已是最新版本"。
func (h *SettingHandler) GetAppUpdateManifest(c *gin.Context) {
	target := strings.TrimSuffix(c.Param("target"), ".json")
	manifest, err := h.settingService.GetAppUpdateManifest(c.Request.Context(), target)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if manifest == nil {
		c.Status(http.StatusNoContent)
		return
	}
	// Tauri updater 读取的是裸 JSON，不能套 {code,data} 外层。
	c.JSON(http.StatusOK, manifest)
}

// GetAppDownloads 返回已发布的完整安装包下载地址（无需认证）
// GET /api/v1/app/downloads
//
// 只包含官网给新用户下载的 DMG / setup EXE，不包含自动更新包和签名。
func (h *SettingHandler) GetAppDownloads(c *gin.Context) {
	downloads, err := h.settingService.GetAppDownloads(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, downloads)
}
