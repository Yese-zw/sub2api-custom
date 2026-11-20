//go:build embed || unit

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicDocumentTitle(t *testing.T) {
	assert.Equal(t, "灵思智域 - AI API 中转", publicDocumentTitle(""))
	assert.Equal(t, "灵思智域 - AI API 中转", publicDocumentTitle("Sub2API"))
	assert.Equal(t, "灵思智域 - AI API 中转", publicDocumentTitle("  sub2api  "))
	assert.Equal(t, "我的站点 - AI API 中转", publicDocumentTitle("我的站点"))
}

func TestApplyRequestSEO(t *testing.T) {
	html := []byte(`<meta name="robots" data-seo="robots" content="index, follow" /><link rel="canonical" data-seo="canonical" href="https://lingsiai.vip/home" /><meta property="og:url" data-seo="canonical" content="https://lingsiai.vip/home" />`)

	home := string(applyRequestSEO(html, "/home"))
	assert.Contains(t, home, `content="index, follow"`)
	assert.Contains(t, home, `href="https://lingsiai.vip/home"`)

	root := string(applyRequestSEO(html, "/"))
	assert.Contains(t, root, `href="https://lingsiai.vip/home"`)

	plaza := string(applyRequestSEO(html, "/model-plaza"))
	assert.Contains(t, plaza, `href="https://lingsiai.vip/model-plaza"`)
	assert.Contains(t, plaza, `content="https://lingsiai.vip/model-plaza"`)
	assert.NotContains(t, plaza, "lingsiai.vip/home")

	admin := string(applyRequestSEO(html, "/admin/dashboard"))
	assert.Contains(t, admin, `content="noindex, nofollow"`)
	assert.NotContains(t, admin, `content="index, follow"`)

	legal := string(applyRequestSEO(html, "/legal/terms"))
	assert.Contains(t, legal, `href="https://lingsiai.vip/legal/terms"`)
	assert.Contains(t, legal, `content="index, follow"`)

	unsafe := string(applyRequestSEO(html, "/legal/../admin"))
	assert.Contains(t, unsafe, `content="noindex, nofollow"`)
}
