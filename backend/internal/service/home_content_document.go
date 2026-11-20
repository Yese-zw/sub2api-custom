package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

const (
	// homeContentInlineLimit keeps short custom HTML on the public settings
	// payload. Larger documents, including full pages with embedded images,
	// are served from a dedicated endpoint so they are not downloaded on
	// every route.
	homeContentInlineLimit = 8 * 1024
	homeContentPath        = "/api/v1/settings/home"
	homeAssetPathPrefix    = "/api/v1/settings/home-asset/"
	homeImageMaxEdge       = 128
	homeImageInlineMax     = 24 * 1024
	homeImageDecodeMax     = 8 << 20
)

// publicHomeContent decides what the anonymous settings payload may carry.
// HTTP(S) values stay inline because the home page iframes them. Large HTML
// is replaced with a same-origin document URL.
func publicHomeContent(raw string) (inline, contentURL string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return trimmed, ""
	}
	if len(trimmed) <= homeContentInlineLimit && !strings.Contains(lower, "data:image/") {
		return trimmed, ""
	}
	return "", homeContentPath
}

type homeAsset struct {
	ContentType string
	Body        []byte
}

var (
	homeAssets  sync.Map
	homeDocs    sync.Map
	homePrepare singleflight.Group
)

// PrepareHomeDocument loads the configured homepage and returns a browser
// document with oversized data-URI images replaced by cached asset URLs.
func (s *SettingService) PrepareHomeDocument(ctx context.Context) (body []byte, etag string, ok bool, err error) {
	raw, err := s.loadRawHomeContent(ctx)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, "", false, err
	}
	if _, contentURL := publicHomeContent(raw); contentURL == "" {
		return nil, "", false, nil
	}
	prepared, err := prepareHomeDocument(raw)
	if err != nil {
		return nil, "", false, err
	}
	sum := sha256.Sum256([]byte(prepared))
	return []byte(prepared), `"` + hex.EncodeToString(sum[:16]) + `"`, true, nil
}

// HomeAsset returns a previously extracted homepage image. A cache miss
// rebuilds the current document once so a restarted process can still serve
// the image URL embedded in that document.
func (s *SettingService) HomeAsset(ctx context.Context, id string) (body []byte, contentType string, ok bool, err error) {
	if asset, found := lookupHomeAsset(id); found {
		return asset.Body, asset.ContentType, true, nil
	}
	if _, _, _, err = s.PrepareHomeDocument(ctx); err != nil {
		return nil, "", false, err
	}
	asset, found := lookupHomeAsset(id)
	if !found {
		return nil, "", false, nil
	}
	return asset.Body, asset.ContentType, true, nil
}

func (s *SettingService) loadRawHomeContent(ctx context.Context) (string, error) {
	if s == nil || s.settingRepo == nil {
		return "", nil
	}
	vals, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyHomeContent})
	if err != nil {
		return "", fmt.Errorf("get home content: %w", err)
	}
	return vals[SettingKeyHomeContent], nil
}

func lookupHomeAsset(id string) (homeAsset, bool) {
	if len(id) != 64 || strings.Trim(id, "0123456789abcdef") != "" {
		return homeAsset{}, false
	}
	value, ok := homeAssets.Load(id)
	if !ok {
		return homeAsset{}, false
	}
	asset, ok := value.(homeAsset)
	return asset, ok
}

func prepareHomeDocument(raw string) (string, error) {
	sum := sha256.Sum256([]byte(raw))
	key := hex.EncodeToString(sum[:])
	if cached, ok := homeDocs.Load(key); ok {
		if doc, ok := cached.(string); ok {
			return doc, nil
		}
	}
	value, err, _ := homePrepare.Do(key, func() (any, error) {
		return externalizeDataImages(raw), nil
	})
	if err != nil {
		return "", err
	}
	doc, _ := value.(string)
	homeDocs.Store(key, doc)
	return doc, nil
}

func externalizeDataImages(html string) string {
	var b strings.Builder
	b.Grow(len(html))
	rest := html
	const marker = "data:image/"
	for {
		i := strings.Index(rest, marker)
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:i])
		rest = rest[i:]
		semi := strings.IndexByte(rest, ';')
		if semi < 0 || semi > 48 || !strings.HasPrefix(rest[semi:], ";base64,") {
			b.WriteString(marker)
			rest = rest[len(marker):]
			continue
		}
		mime := rest[len(marker):semi]
		payloadStart := semi + len(";base64,")
		end := payloadStart
		for end < len(rest) && isBase64Char(rest[end]) {
			end++
		}
		payload := rest[payloadStart:end]
		if replacement, ok := storeDataImage(mime, payload); ok {
			b.WriteString(replacement)
		} else {
			b.WriteString(rest[:end])
		}
		rest = rest[end:]
	}
}

func isBase64Char(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9', b == '+', b == '/', b == '=':
		return true
	default:
		return false
	}
}

func storeDataImage(mime, payload string) (string, bool) {
	if len(payload) < 32 || len(payload) > homeImageDecodeMax {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(payload, "="))
		if err != nil {
			return "", false
		}
	}
	if len(decoded) == 0 || len(decoded) > homeImageDecodeMax {
		return "", false
	}
	body, contentType, external := optimizeHomeImage(mime, decoded)
	if !external {
		return "", false
	}
	sum := sha256.Sum256(body)
	id := hex.EncodeToString(sum[:])
	homeAssets.Store(id, homeAsset{ContentType: contentType, Body: body})
	return homeAssetPathPrefix + id, true
}

func optimizeHomeImage(mime string, decoded []byte) (body []byte, contentType string, external bool) {
	contentType = imageContentType(mime)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(decoded))
	oversized := len(decoded) > homeImageInlineMax || (err == nil && (cfg.Width > homeImageMaxEdge || cfg.Height > homeImageMaxEdge))
	if !oversized {
		return nil, "", false
	}
	if err != nil || (format != "png" && format != "jpeg" && format != "gif") || cfg.Width > 2048 || cfg.Height > 2048 {
		return decoded, contentType, true
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		return decoded, contentType, true
	}
	fitted := fitHomeImage(img, homeImageMaxEdge)
	var buf bytes.Buffer
	if err := png.Encode(&buf, fitted); err != nil || buf.Len() >= len(decoded) {
		return decoded, contentType, true
	}
	return buf.Bytes(), "image/png", true
}

func imageContentType(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "png":
		return "image/png"
	case "jpeg", "jpg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func fitHomeImage(src image.Image, maxEdge int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || (width <= maxEdge && height <= maxEdge) {
		return src
	}
	nw, nh := width, height
	if width >= height {
		nw = maxEdge
		nh = max(1, height*maxEdge/width)
	} else {
		nh = maxEdge
		nw = max(1, width*maxEdge/height)
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := bounds.Min.Y + y*height/nh
		for x := 0; x < nw; x++ {
			sx := bounds.Min.X + x*width/nw
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// Keep image format decoders linked for image.Decode.
var (
	_ = jpeg.Decode
	_ = gif.Decode
	_ = png.Decode
)
