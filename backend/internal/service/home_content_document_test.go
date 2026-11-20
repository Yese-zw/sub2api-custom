package service

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestPublicHomeContentKeepsShortHTMLAndURLsInline(t *testing.T) {
	inline, contentURL := publicHomeContent("  <section>hello</section>  ")
	if inline != "<section>hello</section>" || contentURL != "" {
		t.Fatalf("short html = %q %q", inline, contentURL)
	}
	inline, contentURL = publicHomeContent(" https://example.com/home ")
	if inline != "https://example.com/home" || contentURL != "" {
		t.Fatalf("url = %q %q", inline, contentURL)
	}
}

func TestPublicHomeContentExternalizesLargeHTML(t *testing.T) {
	raw := "<main>" + strings.Repeat("灵思智域", 3000) + "</main>"
	inline, contentURL := publicHomeContent(raw)
	if inline != "" || contentURL != homeContentPath {
		t.Fatalf("large html = %q %q", inline, contentURL)
	}
}

func TestExternalizeOversizedDataImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 360))
	for i := range img.Pix {
		img.Pix[i] = uint8(i)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	original := buf.Bytes()
	html := `<img src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(original) + `" alt="">`
	out := externalizeDataImages(html)
	if strings.Contains(out, "data:image") {
		t.Fatal("data uri was left inline")
	}
	const prefix = homeAssetPathPrefix
	idx := strings.Index(out, prefix)
	if idx < 0 {
		t.Fatalf("asset url missing: %s", out)
	}
	rest := out[idx+len(prefix):]
	end := 0
	for end < len(rest) && ((rest[end] >= '0' && rest[end] <= '9') || (rest[end] >= 'a' && rest[end] <= 'f')) {
		end++
	}
	id := rest[:end]
	if len(id) != 64 {
		t.Fatalf("asset id %q", id)
	}
	asset, ok := lookupHomeAsset(id)
	if !ok {
		t.Fatal("asset was not stored")
	}
	if asset.ContentType != "image/png" {
		t.Fatalf("content type %s", asset.ContentType)
	}
	if len(asset.Body) >= len(original) {
		t.Fatalf("resized image %d is not smaller than %d", len(asset.Body), len(original))
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(asset.Body))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > homeImageMaxEdge || cfg.Height > homeImageMaxEdge {
		t.Fatalf("resized dimensions %dx%d", cfg.Width, cfg.Height)
	}
}

func TestPublicImageValueExternalizesLargeDataURI(t *testing.T) {
	raw := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 4096)
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)

	got := publicImageValue("test_logo", dataURI)
	if !strings.HasPrefix(got, homeAssetPathPrefix) {
		t.Fatalf("large image not externalized: %.60q", got)
	}
	asset, ok := lookupHomeAsset(strings.TrimPrefix(got, homeAssetPathPrefix))
	if !ok || asset.ContentType != "image/png" || !bytes.Equal(asset.Body, raw) {
		t.Fatalf("asset mismatch: ok=%v type=%q len=%d", ok, asset.ContentType, len(asset.Body))
	}
	if again := publicImageValue("test_logo", dataURI); again != got {
		t.Fatalf("cached url changed: %q != %q", again, got)
	}
}

func TestPublicImageValueKeepsSmallURLAndSVGValues(t *testing.T) {
	small := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("tiny-image-bytes"))
	if got := publicImageValue("k1", small); got != small {
		t.Fatalf("small data uri changed: %q", got)
	}
	if got := publicImageValue("k2", "https://cdn.example.com/logo.png"); got != "https://cdn.example.com/logo.png" {
		t.Fatalf("url changed: %q", got)
	}
	svg := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("<svg/>"), 2000))
	if got := publicImageValue("k3", svg); got != svg {
		t.Fatal("svg data uri should stay inline")
	}
}
