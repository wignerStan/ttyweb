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

// StreamMessage represents a single SSE data payload.
type StreamMessage struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index        int    `json:"index"`
		Delta        Delta  `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Delta holds the content fragment in a streaming response.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
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

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// SSE format: lines starting with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Stream end marker
		if data == "[DONE]" {
			return nil
		}

		var msg StreamMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			// Skip malformed lines
			continue
		}

		if len(msg.Choices) > 0 {
			content := msg.Choices[0].Delta.Content
			if content != "" {
				callback(content)
			}
		}
	}
}
