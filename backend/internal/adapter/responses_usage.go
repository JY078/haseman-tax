package adapter

import (
	"encoding/json"
	"unicode/utf8"
)

/** MaxUsageLogResponseBody 写入 usage_logs.response_body 的最大字节长度（截断时保证合法 UTF-8） */
const MaxUsageLogResponseBody = 12000

/** TruncateResponseBody 截断 API 原始 JSON；按字节上限截断并在 UTF-8 字符边界处收尾，避免 PostgreSQL UTF8 报错 */
func TruncateResponseBody(s string) string {
	if len(s) <= MaxUsageLogResponseBody {
		return s
	}
	s = s[:MaxUsageLogResponseBody]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "…(truncated)"
}

/** TruncateRunes 按 Unicode 字符数截断（用于 result 等中文摘要），避免按字节切片破坏 UTF-8 */
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

/** MergeVisionUsage 优先从原始 JSON 解析 token；解析不到时回退到 ResponsesUsage 指针 */
func MergeVisionUsage(raw []byte, usage *ResponsesUsage) (in, out, tot int) {
	in, out, tot = ParseResponsesAPIUsage(raw)
	if in != 0 || out != 0 || tot != 0 {
		return in, out, tot
	}
	if usage != nil {
		return usage.InputTokens, usage.OutputTokens, usage.TotalTokens
	}
	return 0, 0, 0
}

/** ParseChatCompletionsUsage 从 /v1/chat/completions 响应 JSON 中解析 usage（兼容数字为 float64）。 */
func ParseChatCompletionsUsage(raw []byte) (prompt, completion, total int) {
	var root struct {
		Usage map[string]interface{} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &root); err != nil || root.Usage == nil {
		return 0, 0, 0
	}
	u := root.Usage
	prompt = intFromJSONNumber(u["prompt_tokens"])
	completion = intFromJSONNumber(u["completion_tokens"])
	total = intFromJSONNumber(u["total_tokens"])
	if total == 0 && (prompt != 0 || completion != 0) {
		total = prompt + completion
	}
	return prompt, completion, total
}

/** ParseResponsesAPIUsage 从 /v1/responses（或兼容结构）的响应 JSON 中解析用量。
 * 使用顶层 map 解析，避免嵌套 struct 与网关字段差异导致 usage 全为 0。
 * 支持 input_tokens / output_tokens / total_tokens，以及 prompt_tokens / completion_tokens 别名。
 */
func ParseResponsesAPIUsage(raw []byte) (input, output, total int) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0, 0, 0
	}
	uVal, ok := m["usage"]
	if !ok || uVal == nil {
		return 0, 0, 0
	}
	u, ok := uVal.(map[string]interface{})
	if !ok {
		return 0, 0, 0
	}
	input = intFromJSONNumber(u["input_tokens"])
	if input == 0 {
		input = intFromJSONNumber(u["prompt_tokens"])
	}
	output = intFromJSONNumber(u["output_tokens"])
	if output == 0 {
		output = intFromJSONNumber(u["completion_tokens"])
	}
	total = intFromJSONNumber(u["total_tokens"])
	if total == 0 && (input != 0 || output != 0) {
		total = input + output
	}
	return input, output, total
}

func intFromJSONNumber(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}
