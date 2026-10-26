package adapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const ResponsesBaseURL = "https://yunwu.ai"

// ResponseInput represents a message in the responses API
type ResponseInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseRequest is the request body for /v1/responses (GPT-5 and above)
type ResponseRequest struct {
	Model string          `json:"model"`
	Input []ResponseInput `json:"input"`
	// Responses Lite requires parallel tool execution to be explicitly disabled.
	ParallelToolCalls bool `json:"parallel_tool_calls"`
	Stream            bool `json:"stream,omitempty"`
	MaxOutputTokens   int  `json:"max_output_tokens,omitempty"`
}

// ResponseUsage holds token usage from responses API
// The Responses API uses input_tokens/output_tokens (not prompt_tokens/completion_tokens)
type ResponseUsage struct {
	PromptTokens     int `json:"input_tokens"`
	CompletionTokens int `json:"output_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// responseOutput is a single output item from /v1/responses
type responseOutput struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"` // "output_text"
		Text string `json:"text"`
	} `json:"content"`
}

// ResponseResult is the non-streaming response from /v1/responses
type ResponseResult struct {
	ID          string           `json:"id"`
	Object      string           `json:"object"`
	CreatedAt   int64            `json:"created_at"`
	Status      string           `json:"status"`
	Output      []responseOutput `json:"output"`
	Usage       ResponseUsage    `json:"usage"`
	RawResponse string           `json:"-"`
	// Choices is populated after parsing for compatibility with handleChatSync
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"-"`
}

// ResponseAdapter handles chat for GPT-5 and above models using /v1/responses
type ResponseAdapter struct {
	APIKey  string
	BaseURL string
}

func NewResponseAdapter(apiKey string, baseURL string) *ResponseAdapter {
	if baseURL == "" {
		baseURL = ResponsesBaseURL
	}
	return &ResponseAdapter{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// Respond sends a non-streaming request to /v1/responses and returns the result
func (a *ResponseAdapter) Respond(req ResponseRequest) (*ResponseResult, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/v1/responses", bytes.NewBuffer(body))
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

	var result ResponseResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	result.RawResponse = string(respBody)

	inU, outU, totU := ParseResponsesAPIUsage(respBody)
	if inU != 0 || outU != 0 || totU != 0 {
		result.Usage = ResponseUsage{
			PromptTokens:     inU,
			CompletionTokens: outU,
			TotalTokens:      totU,
		}
	}

	// Convert output[] to Choices[] for compatibility with handleChatSync
	for i, out := range result.Output {
		var text string
		for _, part := range out.Content {
			if part.Type == "output_text" {
				text += part.Text
			}
		}
		result.Choices = append(result.Choices, struct {
			Index        int         `json:"index"`
			Message      ChatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{
			Index:        i,
			Message:      ChatMessage{Role: out.Role, Content: text},
			FinishReason: "stop",
		})
	}

	return &result, nil
}

// ResponseStreamCallback is called for each streamed chunk
type ResponseStreamCallback func(content string, done bool, usage *ResponseUsage)

// responseEvent represents a Server-Sent Event from /v1/responses stream
type responseEvent struct {
	Type  string         `json:"type"`
	Delta string         `json:"delta,omitempty"`
	Usage *ResponseUsage `json:"usage,omitempty"`
	// For response.completed event — usage is nested under response.usage
	Response *struct {
		Usage *ResponseUsage `json:"usage,omitempty"`
	} `json:"response,omitempty"`
}

// RespondStream sends a streaming request to /v1/responses
func (a *ResponseAdapter) RespondStream(req ResponseRequest, w io.Writer, cb ResponseStreamCallback) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/v1/responses", bytes.NewBuffer(body))
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
	var lastUsage *ResponseUsage
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		// Track event type from "event:" lines
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			cb("", true, lastUsage)
			break
		}

		var event responseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Handle Responses API event types
		switch {
		case event.Type == "response.output_text.delta" || eventType == "response.output_text.delta":
			if event.Delta != "" {
				cb(event.Delta, false, nil)
			}
		case event.Type == "response.completed" || eventType == "response.completed":
			if event.Response != nil && event.Response.Usage != nil {
				lastUsage = event.Response.Usage
			}
			cb("", true, lastUsage)
			return nil
		case event.Type == "response.done" || eventType == "response.done":
			cb("", true, lastUsage)
			return nil
		}

		// Also handle Chat Completions-style chunks (fallback for proxies)
		if event.Usage != nil {
			lastUsage = event.Usage
		}

		eventType = "" // reset after processing data line
	}

	return scanner.Err()
}
