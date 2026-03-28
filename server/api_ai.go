package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"ttyweb/ai"
)

// aiRequest represents the JSON body for the AI command endpoint.
type aiRequest struct {
	Role   string `json:"role"`
	Prompt string `json:"prompt"`
}

// handleAICommand handles POST /api/ai/command.
// It calls an OpenAI-compatible LLM API to generate a terminal command
// based on the user prompt and the selected AI role.
func (server *Server) handleAICommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body aiRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Prompt == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// Look up the role. Default to "cli-expert" if not specified.
	roleID := body.Role
	if roleID == "" {
		roleID = "cli-expert"
	}
	role := ai.GetRoleDefault(roleID)
	systemPrompt := role.SystemPrompt + "\n\n" + role.Suffix

	// Resolve LLM configuration.
	// Priority: 1) env vars, 2) request headers, 3) query params.
	apiKey := os.Getenv("LLM_API_KEY")
	apiURL := os.Getenv("LLM_API_URL")
	model := os.Getenv("LLM_MODEL")

	if apiKey == "" {
		apiKey = r.Header.Get("X-LLM-Api-Key")
	}
	if apiURL == "" {
		apiURL = r.Header.Get("X-LLM-Api-URL")
	}
	if model == "" {
		model = r.Header.Get("X-LLM-Model")
	}

	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	if apiURL == "" {
		apiURL = r.URL.Query().Get("api_url")
	}
	if model == "" {
		model = r.URL.Query().Get("model")
	}

	// Apply defaults if still not set.
	if apiURL == "" {
		apiURL = "https://api.openai.com"
	}
	if model == "" {
		model = "gpt-4"
	}

	// If no API key is available, return a graceful error.
	if apiKey == "" {
		writeAPISuccess(w, map[string]interface{}{
			"command":     "",
			"explanation": "No LLM API key configured. Set LLM_API_KEY environment variable or pass X-LLM-Api-Key header.",
		})
		return
	}

	// Create client and call the API.
	client := ai.NewClient(apiKey, apiURL, model)
	content, err := client.ChatCompletion(r.Context(), systemPrompt, body.Prompt)
	if err != nil {
		log.Printf("[AI] request failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("AI request failed: %s", err.Error()))
		return
	}

	// Extract the command from the response.
	command := ai.ExtractCommand(content)
	explanation := fmt.Sprintf("[%s] %s", role.ID, model)

	writeAPISuccess(w, map[string]interface{}{
		"command":     command,
		"explanation": explanation,
	})
}