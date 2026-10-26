package adapter

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const ChatCompletionsBaseURL = "https://yunwu.ai"

// ChatMessage represents a message in the chat (text-only)
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// VisionTextPart is a text part in a multimodal message
type VisionTextPart struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// VisionImageURL holds the image URL (base64 data URL or https URL)
type VisionImageURL struct {
	URL string `json:"url"`
}

// VisionImagePart is an image part in a multimodal message
type VisionImagePart struct {
	Type     string         `json:"type"` // "image_url"
	ImageURL VisionImageURL `json:"image_url"`
}

// VisionMessage is a multimodal user message containing both text and images
type VisionMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"` // []VisionTextPart | []VisionImagePart mix
}

// VisionRequest is the request body for a vision chat completion
type VisionRequest struct {
	Model     string          `json:"model"`
	Messages  []VisionMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

// ChatRequest is the request body for /v1/chat/completions
type ChatRequest struct {
	Model         string         `json:"model"`
	Messages      []ChatMessage  `json:"messages"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
	Temperature   float64        `json:"temperature,omitempty"`
}

// StreamOptions enables usage reporting in streaming responses
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatUsage holds token usage info from response
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is a non-streaming response from /v1/chat/completions
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage       ChatUsage `json:"usage"`
	RawResponse string    `json:"-"` // full JSON response body
}

type apiErrorEnvelope struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

// StreamChunk represents a single SSE chunk
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *ChatUsage `json:"usage,omitempty"`
}

// shortTransport is a shared Transport for short-lived API calls (task creation,
// polling). It reuses TCP connections and has the same pool sizing as httpClient
// but shorter per-request timeouts to fail fast on hung upstreams.
var shortTransport = &http.Transport{
	MaxIdleConns:          200,
	MaxIdleConnsPerHost:   50,
	MaxConnsPerHost:       100,
	IdleConnTimeout:       60 * time.Second,
	ResponseHeaderTimeout: 25 * time.Second,
	WriteBufferSize:       8 * 1024,
	ReadBufferSize:        8 * 1024,
	DisableKeepAlives:     false,
}

// shortClient is for task-creation requests (30 s total deadline).
var shortClient = &http.Client{
	Timeout:   30 * time.Second,
	Transport: shortTransport,
}

// pollClient is for lightweight polling requests (15 s total deadline).
var pollClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: shortTransport,
}

const (
	imageRequestTimeout        = 20 * time.Minute
	imageResponseHeaderTimeout = 19 * time.Minute
	chatRequestTimeout         = 10 * time.Minute
	chatResponseHeaderTimeout  = 9 * time.Minute
)

// imageHTTPClient is reserved for long-running image-generation requests.
// Keep it separate from the shared chat client so only image generation gets
// the extended 20-minute deadline.
var imageHTTPClient = &http.Client{
	Timeout: imageRequestTimeout,
	Transport: &http.Transport{
		// Allow upstreams to spend most of the budget before first byte.
		ResponseHeaderTimeout: imageResponseHeaderTimeout,
		// Isolate long-running requests so one HTTP/2 reset cannot fail every task.
		ForceAttemptHTTP2: false,
		TLSNextProto:      make(map[string]func(string, *tls.Conn) http.RoundTripper),

		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     60 * time.Second,
		WriteBufferSize:     8 * 1024,
		ReadBufferSize:      8 * 1024,
		DisableKeepAlives:   false,
	},
}

// httpClient is a shared client tuned for a 2-core 2 GB / 100 Mbps server.
// A single instance is reused across all requests to avoid fd/goroutine leaks.
var httpClient = &http.Client{
	// 10-minute hard deadline covers even the slowest upstream model.
	Timeout: chatRequestTimeout,
	Transport: &http.Transport{
		// Allow up to 9 min for the upstream to send the first response byte
		// (leaves 1 min for body transfer within the 10-min total).
		ResponseHeaderTimeout: chatResponseHeaderTimeout,

		// No upstream rate limiting; most requests are text-only (small payload).
		// Pool sized for realistic concurrency on a 2-core 2 GB host.
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     100,

		// Recycle idle connections quickly to free sockets on a constrained host.
		IdleConnTimeout: 60 * time.Second,

		// Small per-connection I/O buffers reduce heap pressure under load.
		// 8 KB is enough for typical JSON API payloads; increase only if needed.
		WriteBufferSize: 8 * 1024,
		ReadBufferSize:  8 * 1024,

		// Keep TLS sessions alive so re-handshakes don't burn CPU on 2 cores.
		DisableKeepAlives: false,
	},
}

// ChatAdapter handles chat completions for models below GPT-5
type ChatAdapter struct {
	APIKey  string
	BaseURL string
}

func NewChatAdapter(apiKey string, baseURL string) *ChatAdapter {
	if baseURL == "" {
		baseURL = ChatCompletionsBaseURL
	}
	return &ChatAdapter{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// ── Responses API 格式（/v1/responses）──
// 参考：https://yunwu.apifox.cn/api-302363809
// 内容类型与 /v1/chat/completions 不同，使用 input_text / input_image

// ResponsesTextPart 是 /v1/responses 的文本内容块
type ResponsesTextPart struct {
	Type string `json:"type"` // "input_text"
	Text string `json:"text"`
}

// ResponsesImagePart 是 /v1/responses 的图片内容块
// image_url 直接是字符串（data URL 或 https URL），不是嵌套对象
type ResponsesImagePart struct {
	Type     string `json:"type"`      // "input_image"
	ImageURL string `json:"image_url"` // "data:image/png;base64,..."
}

// ResponsesInput 是 /v1/responses 的单条消息
type ResponsesInput struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string 或 []interface{}（多模态）
}

// ResponsesRequest 是 /v1/responses 的请求体
type ResponsesRequest struct {
	Model string           `json:"model"`
	Input []ResponsesInput `json:"input"`
	Tools []ResponsesTool  `json:"tools,omitempty"`
	// Responses Lite requires parallel tool execution to be explicitly disabled.
	ParallelToolCalls bool `json:"parallel_tool_calls"`
	MaxOutputTokens   int  `json:"max_output_tokens,omitempty"`
}

// ResponsesTool describes an official Responses API hosted tool.
type ResponsesTool struct {
	Type              string                `json:"type"`
	SearchContextSize string                `json:"search_context_size,omitempty"`
	Filters           *ResponsesToolFilters `json:"filters,omitempty"`
}

type ResponsesToolFilters struct {
	AllowedDomains []string `json:"allowed_domains,omitempty"`
}

// VisionRequestOptions controls optional hosted tools for a Responses request.
type VisionRequestOptions struct {
	WebSearch         bool
	AllowedDomains    []string
	SearchContextSize string
}

// ResponsesUsage holds token usage info from /v1/responses
type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ResponsesResponse 是 /v1/responses 的响应体结构
// 与 /v1/chat/completions 不同：用 output[].content[].text 而非 choices[].message.content
type ResponsesResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Output []struct {
		Role    string `json:"role"`
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"` // "output_text"
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage ResponsesUsage `json:"usage"`
}

// AnalyzeImageWithVision 使用 POST /v1/responses 格式发送多模态图片分析请求。
// Returns (text, usage, rawResponseBody, error)。rawResponseBody 用于落库与再次解析 usage。
func (a *ChatAdapter) AnalyzeImageWithVision(modelName string, imageData string, mimeType string, prompt string) (string, *ResponsesUsage, []byte, error) {
	return a.AnalyzeImagesWithVision(modelName, []VisionImageItem{{Data: imageData, MimeType: mimeType}}, "", prompt, 2000)
}

func parseAPIErrorEnvelope(respBody []byte) error {
	var envelope apiErrorEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil || envelope.Error == nil {
		return nil
	}
	msg := strings.TrimSpace(envelope.Error.Message)
	if msg == "" {
		msg = "upstream returned error response"
	}
	if envelope.Error.Code != "" {
		return fmt.Errorf("%s (code=%s)", msg, envelope.Error.Code)
	}
	return fmt.Errorf("%s", msg)
}

func (a *ChatAdapter) AnalyzeImageWithVisionChatCompletions(modelName string, imageData string, mimeType string, prompt string) (string, *ResponsesUsage, []byte, error) {
	return a.AnalyzeImagesWithVisionChatCompletions(modelName, []VisionImageItem{{Data: imageData, MimeType: mimeType}}, prompt)
}

func (a *ChatAdapter) AnalyzeImagesWithVisionChatCompletions(modelName string, images []VisionImageItem, prompt string) (string, *ResponsesUsage, []byte, error) {
	content := []interface{}{VisionTextPart{Type: "text", Text: prompt}}
	for _, image := range images {
		mimeType := image.MimeType
		if mimeType == "" {
			mimeType = "image/png"
		}
		content = append(content, VisionImagePart{
			Type:     "image_url",
			ImageURL: VisionImageURL{URL: fmt.Sprintf("data:%s;base64,%s", mimeType, image.Data)},
		})
	}
	req := VisionRequest{
		Model:     modelName,
		MaxTokens: 2000,
		Messages: []VisionMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", nil, nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequest("POST", a.BaseURL+"/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", nil, nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, respBody, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	if err := parseAPIErrorEnvelope(respBody); err != nil {
		return "", nil, respBody, err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", nil, respBody, fmt.Errorf("unmarshal response: %w", err)
	}
	var builder strings.Builder
	for _, choice := range chatResp.Choices {
		if text := strings.TrimSpace(choice.Message.Content); text != "" {
			builder.WriteString(text)
		}
	}
	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", nil, respBody, fmt.Errorf("no message content found in chat completions response")
	}
	usage := &ResponsesUsage{
		InputTokens:  chatResp.Usage.PromptTokens,
		OutputTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:  chatResp.Usage.TotalTokens,
	}
	return text, usage, respBody, nil
}

func extractResponsesText(respBody []byte, r ResponsesResponse) string {
	var builder strings.Builder
	for _, msg := range r.Output {
		for _, part := range msg.Content {
			if (part.Type == "output_text" || part.Type == "text" || part.Type == "") && strings.TrimSpace(part.Text) != "" {
				builder.WriteString(part.Text)
			}
		}
	}
	if text := strings.TrimSpace(builder.String()); text != "" {
		return text
	}

	var root map[string]interface{}
	if err := json.Unmarshal(respBody, &root); err != nil {
		return ""
	}
	for _, key := range []string{"output_text", "text", "content"} {
		if text, ok := root[key].(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	if choices, ok := root["choices"].([]interface{}); ok {
		for _, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				continue
			}
			message, ok := choiceMap["message"].(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := message["content"].(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	if output, ok := root["output"].([]interface{}); ok {
		return extractTextFromOutputItems(output)
	}
	return ""
}

func extractTextFromOutputItems(output []interface{}) string {
	var builder strings.Builder
	for _, item := range output {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if text, ok := itemMap["text"].(string); ok && strings.TrimSpace(text) != "" {
			builder.WriteString(text)
			continue
		}
		if contentText, ok := itemMap["content"].(string); ok && strings.TrimSpace(contentText) != "" {
			builder.WriteString(contentText)
			continue
		}
		content, ok := itemMap["content"].([]interface{})
		if !ok {
			continue
		}
		for _, part := range content {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := partMap["text"].(string); ok && strings.TrimSpace(text) != "" {
				builder.WriteString(text)
				continue
			}
			if text, ok := partMap["content"].(string); ok && strings.TrimSpace(text) != "" {
				builder.WriteString(text)
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

// VisionImageItem holds a single image payload (base64 data + mime type).
type VisionImageItem struct {
	Data     string
	MimeType string
}

func (a *ChatAdapter) AnalyzeImagesWithVision(modelName string, images []VisionImageItem, systemPrompt string, userPrompt string, maxOutputTokens int) (string, *ResponsesUsage, []byte, error) {
	return a.AnalyzeImagesWithVisionContext(context.Background(), modelName, images, systemPrompt, userPrompt, maxOutputTokens)
}

func (a *ChatAdapter) AnalyzeImagesWithVisionContext(ctx context.Context, modelName string, images []VisionImageItem, systemPrompt string, userPrompt string, maxOutputTokens int) (string, *ResponsesUsage, []byte, error) {
	return a.AnalyzeImagesWithVisionContextOptions(ctx, modelName, images, systemPrompt, userPrompt, maxOutputTokens, VisionRequestOptions{})
}

// AnalyzeImagesWithVisionContextOptions sends a Responses request with optional official hosted tools.
func (a *ChatAdapter) AnalyzeImagesWithVisionContextOptions(ctx context.Context, modelName string, images []VisionImageItem, systemPrompt string, userPrompt string, maxOutputTokens int, options VisionRequestOptions) (string, *ResponsesUsage, []byte, error) {
	// Build user content: images first, then the text instruction
	userContent := make([]interface{}, 0, len(images)+1)
	for _, img := range images {
		mt := img.MimeType
		if mt == "" {
			mt = "image/png"
		}
		userContent = append(userContent, ResponsesImagePart{
			Type:     "input_image",
			ImageURL: fmt.Sprintf("data:%s;base64,%s", mt, img.Data),
		})
	}
	userContent = append(userContent, ResponsesTextPart{Type: "input_text", Text: userPrompt})

	inputs := make([]ResponsesInput, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		inputs = append(inputs, ResponsesInput{Role: "system", Content: systemPrompt})
	}
	inputs = append(inputs, ResponsesInput{Role: "user", Content: userContent})

	reqBody := ResponsesRequest{
		Model:           modelName,
		Input:           inputs,
		MaxOutputTokens: maxOutputTokens,
	}
	if options.WebSearch {
		tool := ResponsesTool{Type: "web_search"}
		tool.SearchContextSize = options.SearchContextSize
		if tool.SearchContextSize == "" {
			tool.SearchContextSize = "low"
		}
		if len(options.AllowedDomains) > 0 {
			tool.Filters = &ResponsesToolFilters{AllowedDomains: options.AllowedDomains}
		}
		reqBody.Tools = []ResponsesTool{tool}
	}
	if reqBody.MaxOutputTokens <= 0 {
		reqBody.MaxOutputTokens = 1500
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/v1/responses", bytes.NewBuffer(body))
	if err != nil {
		return "", nil, nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, respBody, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	if err := parseAPIErrorEnvelope(respBody); err != nil {
		return "", nil, respBody, err
	}

	var r ResponsesResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return "", nil, respBody, fmt.Errorf("unmarshal response: %w", err)
	}
	inU, outU, totU := ParseResponsesAPIUsage(respBody)
	if inU != 0 || outU != 0 || totU != 0 {
		r.Usage = ResponsesUsage{InputTokens: inU, OutputTokens: outU, TotalTokens: totU}
	}
	if text := strings.TrimSpace(extractResponsesText(respBody, r)); text != "" {
		return text, &r.Usage, respBody, nil
	}
	return "", &r.Usage, respBody, fmt.Errorf("no output_text in response")
}

// AnalyzeImagesForVariations sends a multi-image vision request to /v1/responses and returns raw text.
// systemPrompt is passed as the first "system" input message; images + userPrompt make up the "user" message.
func (a *ChatAdapter) AnalyzeImagesForVariations(modelName string, images []VisionImageItem, systemPrompt string, userPrompt string) (string, error) {
	text, _, _, err := a.AnalyzeImagesWithVision(modelName, images, systemPrompt, userPrompt, 1500)
	return text, err
}

// Chat sends a non-streaming chat request and returns the full response
func (a *ChatAdapter) Chat(req ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	chatResp.RawResponse = string(respBody)
	if chatResp.Usage.PromptTokens == 0 && chatResp.Usage.CompletionTokens == 0 && chatResp.Usage.TotalTokens == 0 {
		p, c, t := ParseChatCompletionsUsage(respBody)
		if p != 0 || c != 0 || t != 0 {
			chatResp.Usage = ChatUsage{PromptTokens: p, CompletionTokens: c, TotalTokens: t}
		}
	}

	return &chatResp, nil
}

// ChatStreamCallback is called for each streamed chunk of text content
type ChatStreamCallback func(content string, done bool, usage *ChatUsage)

// ChatStream sends a streaming chat request and calls cb for each chunk
func (a *ChatAdapter) ChatStream(req ChatRequest, w io.Writer, cb ChatStreamCallback) error {
	req.Stream = true
	req.StreamOptions = &StreamOptions{IncludeUsage: true}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	// SSE lines rarely exceed a few KB; 64 KB cap prevents runaway allocations
	// on a memory-constrained server while handling edge-case large chunks.
	scanner.Buffer(make([]byte, 32*1024), 64*1024)
	var lastUsage *ChatUsage

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			cb("", true, lastUsage)
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Capture usage from any chunk (including usage-only chunks before [DONE])
		if chunk.Usage != nil {
			lastUsage = chunk.Usage
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if w != nil {
					fmt.Fprintf(w, "data: %s\n\n", data)
				}
				cb(choice.Delta.Content, false, nil)
			}
		}
	}

	return scanner.Err()
}
