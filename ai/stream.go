// Package ai provides OpenAI-compatible chat completion streaming.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Delta holds the content fragment in a streaming response.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Choice is a single completion choice within a StreamMessage.
type Choice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// StreamMessage represents a single SSE data payload.
type StreamMessage struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Choices []Choice `json:"choices"`
}

// ChatRequest is the JSON body sent to the OpenAI-compatible chat completions endpoint.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamChatCompletion sends a chat completion request with streaming enabled
// to the given OpenAI-compatible API URL. The callback is invoked for each
// content delta received from the SSE stream.
//
// apiURL should be the base URL (e.g. "https://api.openai.com/v1").
// apiKey is the Bearer token.
// model is the model identifier (e.g. "gpt-4").
// systemPrompt sets the system role message content.
// userPrompt sets the user role message content.
func StreamChatCompletion(
	ctx context.Context,
	apiURL string,
	apiKey string,
	model string,
	systemPrompt string,
	userPrompt string,
	callback func(token string),
) error {
	endpoint := apiURL + "/chat/completions"

	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return processSSEStream(ctx, resp.Body, callback)
}

// processSSEStream reads SSE events from body, invoking callback for each content delta.
func processSSEStream(ctx context.Context, body io.ReadCloser, callback func(token string)) error {
	defer func() { _ = body.Close() }()
	reader := bufio.NewReader(body)

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("processSSEStream: context cancelled: %w", err)
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			return nil
		}

		if content := extractContent(data); content != "" {
			callback(content)
		}
	}
}

// extractContent parses an SSE data payload and returns the delta content, or "" if none.
func extractContent(data string) string {
	var msg StreamMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return ""
	}
	if len(msg.Choices) > 0 {
		return msg.Choices[0].Delta.Content
	}
	return ""
}
