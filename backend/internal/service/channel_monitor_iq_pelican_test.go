package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractIQHTML(t *testing.T) {
	t.Run("fenced html block", func(t *testing.T) {
		got := extractIQHTML("Here you go:\n```html\n<!DOCTYPE html><html><body></body></html>\n```\nenjoy")
		require.Equal(t, "<!DOCTYPE html><html><body></body></html>", got)
	})
	t.Run("bare svg without fence", func(t *testing.T) {
		got := extractIQHTML("sure <svg width=\"10\"><circle r=\"1\"/></svg> done")
		require.Equal(t, "<svg width=\"10\"><circle r=\"1\"/></svg>", got)
	})
	t.Run("no html", func(t *testing.T) {
		require.Empty(t, extractIQHTML("I cannot draw that."))
	})
}

func TestAssessIQHTML(t *testing.T) {
	shapes := func(n int) string { return strings.Repeat(`<circle cx="1" cy="1" r="1"/>`, n) }
	pad := strings.Repeat(" ", iqPelicanMinBytes)

	q, _ := assessIQHTML("<svg>" + shapes(3) + "</svg>")
	require.Equal(t, "degraded", q)

	q, _ = assessIQHTML("<svg>" + shapes(20) + "<style>@keyframes spin{}</style>" + pad + "</svg>")
	require.Equal(t, iqPelicanQualityNormal, q)

	q, reason := assessIQHTML("<svg>" + shapes(20) + pad + "</svg>")
	require.Equal(t, "suspicious", q)
	require.Contains(t, reason, "缺少动画")

	q, _ = assessIQHTML("<svg>" + shapes(10) + "<animate/>" + pad + "</svg>")
	require.Equal(t, "suspicious", q)
}
