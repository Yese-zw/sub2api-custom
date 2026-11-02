package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadSSEText(t *testing.T) {
	t.Run("openai chat ignores reasoning and role-only chunks", func(t *testing.T) {
		stream := strings.Join([]string{
			`data: {"choices":[{"delta":{"role":"assistant"}}]}`,
			`data: {"choices":[{"delta":{"reasoning_content":"thinking"}}]}`,
			`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
			``,
			`data: {"choices":[{"delta":{"content":"lo"}}]}`,
			`data: [DONE]`,
		}, "\n")
		got, err := readSSEText(strings.NewReader(stream), MonitorProviderOpenAI, MonitorAPIModeChatCompletions)
		require.NoError(t, err)
		require.Equal(t, "Hello", got)
	})

	t.Run("openai responses", func(t *testing.T) {
		stream := "event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"4"}` + "\n" +
			`data: {"type":"response.output_text.delta","delta":"2"}` + "\n" +
			`data: {"type":"response.completed"}`
		got, err := readSSEText(strings.NewReader(stream), MonitorProviderOpenAI, MonitorAPIModeResponses)
		require.NoError(t, err)
		require.Equal(t, "42", got)
	})

	t.Run("anthropic only text deltas", func(t *testing.T) {
		stream := strings.Join([]string{
			`data: {"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"hmm"}}`,
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"ok"}}`,
		}, "\n")
		got, err := readSSEText(strings.NewReader(stream), MonitorProviderAnthropic, "")
		require.NoError(t, err)
		require.Equal(t, "ok", got)
	})

	t.Run("gemini skips thought parts", func(t *testing.T) {
		stream := `data: {"candidates":[{"content":{"parts":[{"text":"t","thought":true},{"text":"hi"}]}}]}`
		got, err := readSSEText(strings.NewReader(stream), MonitorProviderGemini, "")
		require.NoError(t, err)
		require.Equal(t, "hi", got)
	})

	t.Run("stream error event", func(t *testing.T) {
		_, err := readSSEText(strings.NewReader(`data: {"error":{"message":"overloaded"}}`), MonitorProviderOpenAI, MonitorAPIModeChatCompletions)
		require.ErrorContains(t, err, "overloaded")
	})
}
