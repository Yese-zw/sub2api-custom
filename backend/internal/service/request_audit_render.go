package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// RequestAuditCapture 网关请求路径交给审计服务的原始抓取结果。
// 请求路径上只做引用/移交，不做任何解析与拷贝；渲染（base64 折叠、解压、
// UTF-8 清洗、截断）在审计写入协程中完成，不占用客户端请求的完成时间。
type RequestAuditCapture struct {
	Log *RequestAuditLog

	// RequestRaw 通常直接引用 PrereadBody 的不可变切片（已解压、零拷贝）。
	RequestRaw      []byte
	RequestOverflow bool
	// RequestEncoding 非空表示 RequestRaw 是未解压的压缩字节（兜底抓取路径）。
	RequestEncoding string

	ResponseRaw      []byte
	ResponseOverflow bool
	ResponseEncoding string

	MaxBodyBytes int
}

func (c *RequestAuditCapture) approxBytes() int64 {
	if c == nil {
		return 0
	}
	return int64(len(c.RequestRaw) + len(c.ResponseRaw) + 1024)
}

// requestAuditBase64Patterns 匹配内联 base64 数据（图片/音频等），入库前折叠为占位符，
// 避免几 MB 的图片把文本内容挤出截断上限。
var requestAuditBase64Patterns = []*regexp.Regexp{
	// data URI：data:image/png;base64,....
	regexp.MustCompile(`(data:[a-zA-Z0-9.+/-]{1,100};base64,)[A-Za-z0-9+/=\\]{256,}`),
	// JSON 字段：OpenAI images b64_json、Gemini inlineData.data 等。
	regexp.MustCompile(`("(?:b64_json|data|image_base64|partial_image_b64)"\s*:\s*")[A-Za-z0-9+/=\\]{1000,}`),
}

// renderRequestAuditCapture 把原始抓取渲染为可入库的 RequestAuditLog。
func renderRequestAuditCapture(c *RequestAuditCapture) *RequestAuditLog {
	if c == nil || c.Log == nil {
		return nil
	}
	l := c.Log
	max := c.MaxBodyBytes
	if max <= 0 {
		max = RequestAuditDefaultMaxBodyBytes
	}
	l.RequestBody, l.RequestTruncated = renderRequestAuditBody(c.RequestRaw, c.RequestOverflow, l.RequestContentType, c.RequestEncoding, max)
	l.ResponseBody, l.ResponseTruncated = renderRequestAuditBody(c.ResponseRaw, c.ResponseOverflow, l.ResponseContentType, c.ResponseEncoding, max)
	return l
}

func renderRequestAuditBody(raw []byte, overflow bool, contentType, contentEncoding string, maxBytes int) (string, bool) {
	if len(raw) == 0 {
		return "", overflow
	}
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "multipart/") {
		return fmt.Sprintf("[multipart body, %d bytes, not stored]", len(raw)), overflow
	}
	enc := strings.ToLower(strings.TrimSpace(contentEncoding))
	if enc != "" && enc != "identity" {
		decoded := false
		if enc == "gzip" && !overflow {
			if zr, err := gzip.NewReader(bytes.NewReader(raw)); err == nil {
				if out, err := io.ReadAll(io.LimitReader(zr, int64(maxBytes)*4)); err == nil {
					raw = out
					decoded = true
				}
				_ = zr.Close()
			}
		}
		if !decoded {
			return fmt.Sprintf("[%s-encoded body, %d bytes, not stored]", enc, len(raw)), overflow
		}
	}
	for _, re := range requestAuditBase64Patterns {
		if !re.Match(raw) {
			continue
		}
		raw = re.ReplaceAllFunc(raw, func(m []byte) []byte {
			sub := re.FindSubmatchIndex(m)
			prefixEnd := len(m)
			if len(sub) >= 4 && sub[3] > 0 {
				prefixEnd = sub[3]
			}
			return []byte(fmt.Sprintf("%s[base64 omitted, %d bytes]", m[:prefixEnd], len(m)-prefixEnd))
		})
	}
	truncated := overflow
	if len(raw) > maxBytes {
		raw = raw[:maxBytes]
		truncated = true
	}
	// 截断可能切断多字节字符；尾部残缺不算二进制。
	probe := raw
	if len(probe) > 4096 {
		probe = probe[:4096]
	}
	for i := 0; i < 3 && len(probe) > 0 && !utf8.Valid(probe); i++ {
		probe = probe[:len(probe)-1]
	}
	if !utf8.Valid(probe) {
		return fmt.Sprintf("[binary body (%s), %d bytes, not stored]", contentType, len(raw)), truncated
	}
	s := strings.ToValidUTF8(string(raw), "")
	s = strings.ReplaceAll(s, "\x00", "")
	return s, truncated
}
