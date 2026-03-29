package server

import (
	"encoding/json"
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

// resolveEnvConfig returns the value of the given environment variable, or the
// provided fallback if the variable is empty. Configuration is read exclusively
// from the server environment -- never from HTTP headers or query parameters --
// to prevent SSRF and credential-exposure attacks.
func resolveEnvConfig(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
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

	// Resolve LLM configuration from environment variables only.
	// Headers and query parameters are intentionally ignored to prevent SSRF.
	apiKey := os.Getenv("LLM_API_KEY")
	apiURL := resolveEnvConfig("LLM_API_URL", defaultAPIURL)
	model := resolveEnvConfig("LLM_MODEL", defaultModel)

	if apiKey == "" {
		writeAPISuccess(w, aiCommandResponse{
			Command:     "",
			Explanation: "No LLM API key configured. Set the LLM_API_KEY environment variable.",
		})
		return
	}

	client := ai.NewClient(apiKey, apiURL, model)
	content, err := client.ChatCompletion(r.Context(), systemPrompt, body.Prompt)
	if err != nil {
		log.Printf("[AI] request failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "AI request failed")
		return
	}

	writeAPISuccess(w, aiCommandResponse{
		Command:     ai.ExtractCommand(content),
		Explanation: role.ID,
	})
}
