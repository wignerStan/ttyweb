package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"ttyweb/ai"
)

// aiCommandRequest represents the JSON body for the AI command endpoint.
type aiCommandRequest struct {
	Role   string `json:"role"`
	Prompt string `json:"prompt"`
}

// aiCommandResponse represents the JSON response for the AI command endpoint.
type aiCommandResponse struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
}

// resolveConfig returns the first non-empty value from the given sources.
func resolveConfig(envKey, headerName, queryParam string, r *http.Request) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if v := r.Header.Get(headerName); v != "" {
		return v
	}
	return r.URL.Query().Get(queryParam)
}

// defaultAPIURL is the fallback API URL when none is configured.
const defaultAPIURL = "https://api.openai.com"

// defaultModel is the fallback model when none is configured.
const defaultModel = "gpt-4"

// handleAICommand handles POST /api/ai/command.
// It calls an OpenAI-compatible LLM API to generate a terminal command
// based on the user prompt and the selected AI role.
func (server *Server) handleAICommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body aiCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Prompt == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	role := ai.GetRoleDefault(body.Role)
	systemPrompt := role.SystemPrompt + "\n\n" + role.Suffix

	// Resolve LLM configuration. Priority: env vars > request headers > query params.
	apiKey := resolveConfig("LLM_API_KEY", "X-LLM-Api-Key", "api_key", r)
	apiURL := resolveConfig("LLM_API_URL", "X-LLM-Api-URL", "api_url", r)
	model := resolveConfig("LLM_MODEL", "X-LLM-Model", "model", r)

	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	if model == "" {
		model = defaultModel
	}

	if apiKey == "" {
		writeAPISuccess(w, aiCommandResponse{
			Command:     "",
			Explanation: "No LLM API key configured. Set LLM_API_KEY environment variable or pass X-LLM-Api-Key header.",
		})
		return
	}

	client := ai.NewClient(apiKey, apiURL, model)
	content, err := client.ChatCompletion(r.Context(), systemPrompt, body.Prompt)
	if err != nil {
		log.Printf("[AI] request failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("AI request failed: %s", err.Error()))
		return
	}

	writeAPISuccess(w, aiCommandResponse{
		Command:     ai.ExtractCommand(content),
		Explanation: fmt.Sprintf("[%s] %s", role.ID, model),
	})
}