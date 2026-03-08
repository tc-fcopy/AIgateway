package token

import "testing"

func TestParseOpenAIResponse(t *testing.T) {
	body := []byte(`{"id":"1","usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`)
	usage, err := ParseOpenAIResponse(body)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if usage.TotalTokens != 30 {
		t.Fatalf("expected total tokens 30, got %d", usage.TotalTokens)
	}
}

func TestMergeUsage(t *testing.T) {
	u1 := &TokenUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}
	u2 := &TokenUsage{PromptTokens: 4, CompletionTokens: 5, TotalTokens: 9}
	out := MergeUsage(u1, u2)
	if out.PromptTokens != 5 || out.CompletionTokens != 7 || out.TotalTokens != 12 {
		t.Fatalf("unexpected merged usage: %+v", out)
	}
}
