//go:build embed || unit

package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// gzipMinSize skips compression when the payload is already smaller than a
// typical gzip header plus the bytes it could save.
const gzipMinSize = 256

type gzipCacheEntry struct {
	body []byte
	ok   bool
}

// distGzipCache stores gzipped copies of immutable embedded assets.
var distGzipCache sync.Map

func acceptsGzip(r *http.Request) bool {
	if r == nil || r.Header.Get("Range") != "" {
		return false
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc := strings.TrimSpace(part)
		if i := strings.IndexByte(enc, ';'); i >= 0 {
			enc = strings.TrimSpace(enc[:i])
		}
		if enc == "gzip" {
			return true
		}
	}
	return false
}

func compressibleAssetPath(cleanPath string) bool {
	switch strings.ToLower(path.Ext(cleanPath)) {
	case ".js", ".mjs", ".css", ".svg", ".json", ".html", ".txt", ".xml", ".map":
		return true
	default:
		return false
	}
}

func gzipPayload(raw []byte) ([]byte, bool) {
	if len(raw) < gzipMinSize {
		return nil, false
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, false
	}
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return nil, false
	}
	if err := zw.Close(); err != nil {
		return nil, false
	}
	if buf.Len() >= len(raw) {
		return nil, false
	}
	return buf.Bytes(), true
}

func cachedGzipPayload(key string, raw []byte) ([]byte, bool) {
	if cached, ok := distGzipCache.Load(key); ok {
		entry := cached.(gzipCacheEntry)
		return entry.body, entry.ok
	}
	body, ok := gzipPayload(raw)
	distGzipCache.Store(key, gzipCacheEntry{body: body, ok: ok})
	return body, ok
}

func writeEncodedBody(c *gin.Context, contentType string, body []byte) {
	header := c.Writer.Header()
	header.Add("Vary", "Accept-Encoding")
	if acceptsGzip(c.Request) {
		if gz, ok := gzipPayload(body); ok {
			header.Set("Content-Encoding", "gzip")
			c.Data(http.StatusOK, contentType, gz)
			return
		}
	}
	c.Data(http.StatusOK, contentType, body)
}

func serveDistFile(c *gin.Context, fsys fs.FS, fileServer http.Handler, cleanPath string) {
	applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
	if compressibleAssetPath(cleanPath) {
		c.Header("Vary", "Accept-Encoding")
	}
	if acceptsGzip(c.Request) && compressibleAssetPath(cleanPath) && writeCachedGzipFile(c, fsys, cleanPath) {
		c.Abort()
		return
	}
	fileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func writeCachedGzipFile(c *gin.Context, fsys fs.FS, cleanPath string) bool {
	file, err := fsys.Open(cleanPath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return false
	}
	gz, ok := cachedGzipPayload(cleanPath, raw)
	if !ok {
		return false
	}
	contentType := mime.TypeByExtension(path.Ext(cleanPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Encoding", "gzip")
	c.Data(http.StatusOK, contentType, gz)
	return true
}
