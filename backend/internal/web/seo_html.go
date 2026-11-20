//go:build embed || unit

package web

import (
	"bytes"
	"regexp"
	"strings"
)

const (
	seoSiteOrigin   = "https://lingsiai.vip"
	seoHomeURL      = seoSiteOrigin + "/home"
	seoRobotsIndex  = `data-seo="robots" content="index, follow"`
	seoRobotsHidden = `data-seo="robots" content="noindex, nofollow"`
	defaultSEOTitle = "灵思智域 - AI API 中转"
)

var legalSlugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func publicDocumentTitle(siteName string) string {
	name := strings.TrimSpace(siteName)
	if name == "" || strings.EqualFold(name, "Sub2API") {
		return defaultSEOTitle
	}
	return name + " - AI API 中转"
}

func applyRequestSEO(html []byte, requestPath string) []byte {
	path := requestPath
	if path == "" {
		path = "/"
	}
	if !isIndexablePublicPath(path) {
		return bytes.ReplaceAll(html, []byte(seoRobotsIndex), []byte(seoRobotsHidden))
	}
	canonical := canonicalURL(path)
	if canonical == seoHomeURL {
		return html
	}
	html = bytes.ReplaceAll(html,
		[]byte(`data-seo="canonical" href="`+seoHomeURL+`"`),
		[]byte(`data-seo="canonical" href="`+canonical+`"`),
	)
	return bytes.ReplaceAll(html,
		[]byte(`data-seo="canonical" content="`+seoHomeURL+`"`),
		[]byte(`data-seo="canonical" content="`+canonical+`"`),
	)
}

func isIndexablePublicPath(path string) bool {
	switch path {
	case "/", "/home", "/model-plaza":
		return true
	default:
		slug, ok := strings.CutPrefix(path, "/legal/")
		return ok && legalSlugPattern.MatchString(slug)
	}
}

func canonicalURL(path string) string {
	switch path {
	case "/", "/home":
		return seoHomeURL
	case "/model-plaza":
		return seoSiteOrigin + "/model-plaza"
	default:
		if slug, ok := strings.CutPrefix(path, "/legal/"); ok && legalSlugPattern.MatchString(slug) {
			return seoSiteOrigin + "/legal/" + slug
		}
		return seoHomeURL
	}
}
