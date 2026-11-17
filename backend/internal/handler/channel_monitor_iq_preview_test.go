package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServeChannelMonitorIQPreviewAllowsInlineAnimationScripts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/channel-monitor-iq-preview", ServeChannelMonitorIQPreview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/channel-monitor-iq-preview", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'unsafe-inline'") || strings.Contains(csp, "nonce-") {
		t.Fatalf("preview CSP = %s", csp)
	}
	if recorder.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %s", recorder.Header().Get("X-Frame-Options"))
	}
	if !strings.Contains(recorder.Body.String(), "iq-preview") {
		t.Fatal("preview shell missing message handler")
	}
}
