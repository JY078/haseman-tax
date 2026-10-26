package adapter

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseResponsesAPIUsage_sampleJSON(t *testing.T) {
	raw := `{"background":false,"model":"gpt-4.1-2025-04-14","object":"response","output":[{"content":[{"text":"hi","type":"output_text"}],"role":"assistant","type":"message"}],"usage":{"input_tokens":915,"input_tokens_details":{"cached_tokens":0},"output_tokens":218,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":1133}}`
	in, out, tot := ParseResponsesAPIUsage([]byte(raw))
	if in != 915 || out != 218 || tot != 1133 {
		t.Fatalf("got in=%d out=%d tot=%d", in, out, tot)
	}
}

func TestExtractResponsesText_standardOutputText(t *testing.T) {
	raw := []byte(`{"output":[{"content":[{"text":"标准文本","type":"output_text"}],"role":"assistant","type":"message"}]}`)
	var resp ResponsesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	if got := extractResponsesText(raw, resp); got != "标准文本" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractResponsesText_topLevelOutputText(t *testing.T) {
	raw := []byte(`{"output_text":"顶层文本"}`)
	if got := extractResponsesText(raw, ResponsesResponse{}); got != "顶层文本" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractResponsesText_chatCompletionsCompatible(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"content":"兼容文本"}}]}`)
	if got := extractResponsesText(raw, ResponsesResponse{}); got != "兼容文本" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateResponseBody_validUTF8(t *testing.T) {
	// 截断点落在「中」的 UTF-8 字节中间时，须回退到合法边界
	pad := strings.Repeat("a", MaxUsageLogResponseBody-1)
	s := pad + "中"
	out := TruncateResponseBody(s)
	if !utf8.ValidString(out) {
		t.Fatal("TruncateResponseBody produced invalid UTF-8")
	}
}

func TestTruncateRunes_chinese(t *testing.T) {
	s := strings.Repeat("你", 300)
	out := TruncateRunes(s, 100)
	r := []rune(out)
	if len(r) != 101 {
		t.Fatalf("want 101 runes (100+…), got %d", len(r))
	}
}
