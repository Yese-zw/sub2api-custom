package service

import "testing"

func TestIQAnswerPasses(t *testing.T) {
	cases := map[string]bool{
		"答案是 21":     true,
		"最少取21个":     true,
		"121":        false,
		"210":        false,
		"3.21":       false,
		"21.5":       false,
		"结果为 21。":    true,
		"22":         false,
		"":           false,
		"**21**":     true,
		"总数为 21, 因此": true,
		"21.":        true,
	}
	for in, want := range cases {
		if got := iqAnswerPasses(in); got != want {
			t.Errorf("iqAnswerPasses(%q)=%v want %v", in, got, want)
		}
	}
}

func TestExtractIQAnswer(t *testing.T) {
	resp := `{"output":[{"type":"reasoning"},{"type":"message","content":[{"type":"output_text","text":"答案 21"}]}],"usage":{"input_tokens":5,"output_tokens":7}}`
	if text, in, out, _ := extractIQAnswer([]byte(resp)); text != "答案 21" || in != 5 || out != 7 {
		t.Fatalf("responses: %q %d %d", text, in, out)
	}
	chat := `{"choices":[{"message":{"content":"21"}}]}`
	if text, _, _, _ := extractIQAnswer([]byte(chat)); text != "21" {
		t.Fatalf("chat: %q", text)
	}
	anth := `{"content":[{"type":"text","text":"是 21"}],"usage":{"input_tokens":1,"output_tokens":2}}`
	if text, in, out, _ := extractIQAnswer([]byte(anth)); text != "是 21" || in != 1 || out != 2 {
		t.Fatalf("anthropic: %q", text)
	}
	sse := "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"答案\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"21\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":3,\"output_tokens\":4}}}\n\n"
	if text, in, out, _ := extractIQAnswer([]byte(sse)); text != "答案21" || in != 3 || out != 4 {
		t.Fatalf("sse: %q %d %d", text, in, out)
	}
	if _, _, _, msg := extractIQAnswer([]byte(`{"error":{"message":"bad"}}`)); msg != "bad" {
		t.Fatalf("err: %q", msg)
	}
}
