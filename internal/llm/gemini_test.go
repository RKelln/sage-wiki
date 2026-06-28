package llm

import (
	"strings"
	"testing"
)

// TestGeminiParseResponse_MaxTokensEmpty verifies the MAX_TOKENS empty-content
// branch surfaces FinishReason + output tokens so EmptyContentDetails can emit
// the actionable hint instead of a bare "empty content" message.
func TestGeminiParseResponse_MaxTokensEmpty(t *testing.T) {
	body := []byte(`{
		"candidates": [{"content": {"parts": []}, "finishReason": "MAX_TOKENS"}],
		"usageMetadata": {"promptTokenCount": 1000, "candidatesTokenCount": 800, "totalTokenCount": 1800},
		"modelVersion": "gemini-x"
	}`)
	p := newGeminiProvider("k", "")
	resp, err := p.ParseResponse(body)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
	if resp.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want length (MAX_TOKENS normalized)", resp.FinishReason)
	}
	if resp.Usage.OutputTokens != 800 {
		t.Errorf("OutputTokens = %d, want 800", resp.Usage.OutputTokens)
	}
	details := resp.EmptyContentDetails()
	for _, want := range []string{"finish_reason=length", "summary_max_tokens"} {
		if !strings.Contains(details, want) {
			t.Errorf("EmptyContentDetails missing %q\nfull: %s", want, details)
		}
	}
}

func TestGeminiParseResponse_Normal(t *testing.T) {
	body := []byte(`{
		"candidates": [{"content": {"parts": [{"text": "hi"}]}, "finishReason": "STOP"}],
		"usageMetadata": {"totalTokenCount": 10},
		"modelVersion": "g"
	}`)
	p := newGeminiProvider("k", "")
	resp, err := p.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" {
		t.Errorf("Content = %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop (STOP normalized)", resp.FinishReason)
	}
}
