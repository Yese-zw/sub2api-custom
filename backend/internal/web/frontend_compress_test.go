//go:build unit

package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptsGzip(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/assets/index-AbCd1234.js", nil)
	assert.False(t, acceptsGzip(req))

	req.Header.Set("Accept-Encoding", "deflate, gzip")
	assert.True(t, acceptsGzip(req))

	req.Header.Set("Range", "bytes=0-100")
	assert.False(t, acceptsGzip(req))
}

func TestCompressibleAssetPath(t *testing.T) {
	t.Parallel()

	assert.True(t, compressibleAssetPath("assets/index-AbCd1234.js"))
	assert.True(t, compressibleAssetPath("assets/index-AbCd1234.css"))
	assert.True(t, compressibleAssetPath("logo.svg"))
	assert.False(t, compressibleAssetPath("logo.png"))
	assert.False(t, compressibleAssetPath("assets/font-AbCd1234.woff2"))
}

func TestGzipPayloadRoundTrip(t *testing.T) {
	t.Parallel()

	raw := bytes.Repeat([]byte("function bootstrap(){return 1}"), 40)
	gz, ok := gzipPayload(raw)
	require.True(t, ok)
	assert.Less(t, len(gz), len(raw))

	reader, err := gzip.NewReader(bytes.NewReader(gz))
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, raw, decoded)

	_, ok = gzipPayload([]byte("tiny"))
	assert.False(t, ok)
}
