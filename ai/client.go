package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the maximum time to wait for an LLM API response.
const defaultTimeout = 60 * time.Second

// chatRequest represents the JSON body sent to an OpenAI-compatible API.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage represents a single message in the conversation.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the JSON response from an OpenAI-compatible API.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *chatError   `json:"error,omitempty"`
}

// chatChoice represents a single choice in the API response.
type chatChoice struct {
	Message chatMessage `json:"message"`
}

// chatError represents an error returned by the API.
type chatError struct {
	Message string `json:"message"`
}

// Client wraps an OpenAI-compatible chat completion API.
type Client struct {
	apiKey string
	apiURL string
	model  string
	client *http.Client
}

// NewClient creates a new AI client with the given configuration.
// apiURL should be the base URL of the API (e.g., "https://api.openai.com").
// The /v1/chat/completions path is appended automatically.
func NewClient(apiKey, apiURL, model string) *Client {
	// Normalize: strip trailing slash.
	apiURL = strings.TrimRight(apiURL, "/")

	return &Client{
		apiKey: apiKey,
		apiURL: apiURL,
		model:  model,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// ChatCompletion sends a chat completion request and returns the assistant's response.
// It uses non-streaming mode for simplicity, parsing the full response.
func (c *Client) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.apiURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("API returned empty content")
	}

	return content, nil
}

// ExtractCommand pulls a command from an LLM response.
// If the response contains a markdown code fence (```...```), the content
// inside the fence is returned. Otherwise, the full text is returned.
func ExtractCommand(content string) string {
	// Try to match the first code fence block.
	content = strings.TrimSpace(content)

	fenceStart := strings.Index(content, "```")
	if fenceStart == -1 {
		return content
	}

	// Skip past the opening fence and optional language tag.
	afterFence := content[fenceStart+3:]
	nlIdx := strings.Index(afterFence, "\n")
	if nlIdx == -1 {
		return content
	}
	codeStart := nlIdx + 1

	fenceEnd := strings.Index(afterFence[codeStart:], "```")
	if fenceEnd == -1 {
		// No closing fence — return everything after the opening fence.
		return strings.TrimSpace(afterFence[nlIdx+1:])
	}

	code := afterFence[codeStart : codeStart+fenceEnd]
	return strings.TrimSpace(code)
}
