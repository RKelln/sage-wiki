package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/xoai/sage-wiki/internal/memory"
)

func TestTruncateResultContent_ShortNoop(t *testing.T) {
	content := "a short article body"
	got := truncateResultContent(content, "wiki/concepts/x.md", 2000)
	if got != content {
		t.Errorf("short content should be returned unchanged, got %q", got)
	}
}

func TestTruncateResultContent_ExactCapNoop(t *testing.T) {
	content := strings.Repeat("a", 100)
	got := truncateResultContent(content, "", 100)
	if got != content {
		t.Errorf("content exactly at cap should not be truncated, got len %d", len(got))
	}
}

// Pins the rune-boundary gotcha: truncating multibyte (CJK) content at a byte
// offset would split a rune and emit U+FFFD. The cut must land on a rune start.
func TestTruncateResultContent_RuneSafe(t *testing.T) {
	content := strings.Repeat("世", 100) // 3 bytes each
	got := truncateResultContent(content, "", 10)

	if !utf8.ValidString(got) {
		t.Fatalf("truncated content is not valid UTF-8: %q", got)
	}
	if strings.ContainsRune(got, '�') {
		t.Fatalf("truncated content contains U+FFFD replacement char: %q", got)
	}
	// The body before the marker must be exactly 10 full runes.
	body := got
	if idx := strings.Index(got, "\n\n["); idx >= 0 {
		body = got[:idx]
	}
	if n := utf8.RuneCountInString(body); n != 10 {
		t.Errorf("expected 10 runes before marker, got %d", n)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("expected a truncation marker, got %q", got)
	}
}

func TestTruncateResultContent_MarkerHasReadHint(t *testing.T) {
	content := strings.Repeat("x", 5000)
	got := truncateResultContent(content, "wiki/concepts/attention.md", 100)
	if !strings.Contains(got, `wiki_read("wiki/concepts/attention.md")`) {
		t.Errorf("marker should point at wiki_read with the article path, got %q", got[len(got)-120:])
	}
}

func TestTruncateResultContent_NoPathNoReadHint(t *testing.T) {
	content := strings.Repeat("x", 5000)
	got := truncateResultContent(content, "", 100)
	if strings.Contains(got, "wiki_read(") {
		t.Errorf("marker should omit wiki_read hint when no path, got tail %q", got[len(got)-120:])
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("expected a truncation marker even without a path, got %q", got)
	}
}

// End-to-end: a long entry returned by wiki_search must come back bounded and
// valid (not the full document), so it can't overflow the caller's context.
func TestHandleSearch_TruncatesLongContent(t *testing.T) {
	dir := setupTestProject(t)
	srv, _ := NewServer(dir)
	defer srv.Close()

	// "attention" matches the query; the rest is a large CJK filler body.
	long := "attention " + strings.Repeat("世界 ", 4000)
	srv.mem.Add(memory.Entry{
		ID:          "big",
		Content:     long,
		Tags:        []string{"attention"},
		ArticlePath: "wiki/concepts/attention.md",
	})

	result, err := srv.handleSearch(context.Background(), makeToolRequest(map[string]any{
		"query": "attention",
		"limit": float64(5),
	}))
	if err != nil {
		t.Fatalf("handleSearch: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(mcp.TextContent).Text)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if !utf8.ValidString(text) {
		t.Fatalf("response is not valid UTF-8")
	}
	var resp struct {
		Results []struct {
			ID      string `json:"ID"`
			Content string `json:"Content"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}

	var found bool
	for _, r := range resp.Results {
		if r.ID == "big" {
			found = true
			if utf8.RuneCountInString(r.Content) >= utf8.RuneCountInString(long) {
				t.Errorf("expected truncated content, got %d runes (original %d)",
					utf8.RuneCountInString(r.Content), utf8.RuneCountInString(long))
			}
			if !strings.Contains(r.Content, "truncated") {
				t.Errorf("expected truncation marker in returned content")
			}
		}
	}
	if !found {
		t.Fatalf("entry 'big' not in results")
	}
}
