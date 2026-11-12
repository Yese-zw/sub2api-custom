package service

import (
	"strings"
	"testing"

	judgerassets "github.com/Wei-Shaw/sub2api/internal/pkg/htmljudger/assets"
	"github.com/Wei-Shaw/sub2api/internal/pkg/htmljudger/judge"

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

func TestIQBankPromptAt(t *testing.T) {
	prompts := []string{"a", "b", "c"}
	got, err := iqBankPromptAt(prompts, 0)
	require.NoError(t, err)
	require.Equal(t, "a", got)
	got, err = iqBankPromptAt(prompts, 3)
	require.NoError(t, err)
	require.Equal(t, "a", got)
	got, err = iqBankPromptAt(prompts, 4)
	require.NoError(t, err)
	require.Equal(t, "b", got)
	_, err = iqBankPromptAt(nil, 0)
	require.Error(t, err)
}

func TestJudgeIQHTMLUsesBank(t *testing.T) {
	j, err := judge.Load(judgerassets.FS)
	require.NoError(t, err)
	prompts := j.BankPrompts()
	require.Len(t, prompts, 200)
	require.Contains(t, prompts[0], "鹈鹕")
	require.True(t, j.InBank(prompts[0]))
	require.False(t, j.InBank(iqCreativePrompt))

	quality, reason, err := judgeIQHTML(prompts[0], "<svg><circle r=\"1\"/></svg>")
	require.NoError(t, err)
	require.Contains(t, []string{"normal", "degraded", "borderline"}, quality)
	require.Contains(t, reason, quality)
	require.True(t, strings.Contains(reason, "v6_pool") || strings.Contains(reason, "v6_bank"))

	off, offReason, err := judgeIQHTML(iqCreativePrompt, "<svg><circle r=\"1\"/></svg>")
	require.NoError(t, err)
	require.Contains(t, []string{"normal", "degraded"}, off)
	require.Contains(t, offReason, "raw_v5b")
}
