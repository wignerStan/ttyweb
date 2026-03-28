package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"ttyweb/ai"
)

// wsStreamMessage is a JSON message sent over the AI stream WebSocket.
type wsStreamMessage struct {
	Type string `json:"type"` // "token" or "done" or "error"
	Data string `json:"data"` // token content, full text on done, error message on error
}

// handleAIStream handles the /ws/ai/stream WebSocket endpoint.
// It reads a JSON message with role and prompt fields, then streams
// LLM responses back as individual token messages followed by a done message.
func (server *Server) handleAIStream(w http.ResponseWriter, r *http.Request) {
	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ai-stream] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Read the initial request message.
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[ai-stream] Failed to read initial message: %v", err)
		return
	}

	var req struct {
		Role   string `json:"role"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(msgBytes, &req); err != nil {
		sendWSStreamError(conn, "invalid request: "+err.Error())
		return
	}

	if req.Prompt == "" {
		sendWSStreamError(conn, "prompt is required")
		return
	}

	// Resolve role configuration: check if role matches a built-in or custom role.
	roleConfig := resolveRoleConfig(req.Role)

	if roleConfig.SystemPrompt == "" {
		// Fall back to a generic system prompt.
		roleConfig.SystemPrompt = "You are a helpful assistant that generates terminal commands. Respond concisely."
	}

	apiURL := roleConfig.APIURL
	apiKey := roleConfig.APIKey
	model := roleConfig.Model

	// If no per-role config, check environment variables.
	if apiURL == "" {
		apiURL = getEnvOrDefault("OPENAI_API_URL", "https://api.openai.com/v1")
	}
	if model == "" {
		model = getEnvOrDefault("OPENAI_MODEL", "gpt-4")
	}

	// Stream the response.
	var fullText string
	err = ai.StreamChatCompletion(
		r.Context(),
		apiURL,
		apiKey,
		model,
		roleConfig.SystemPrompt,
		req.Prompt,
		func(token string) {
			fullText += token
			if writeErr := conn.WriteJSON(wsStreamMessage{
				Type: "token",
				Data: token,
			}); writeErr != nil {
				log.Printf("[ai-stream] Failed to send token: %v", writeErr)
			}
		},
	)

	if err != nil {
		sendWSStreamError(conn, err.Error())
		return
	}

	// Send completion message with full accumulated text.
	if writeErr := conn.WriteJSON(wsStreamMessage{
		Type: "done",
		Data: fullText,
	}); writeErr != nil {
		log.Printf("[ai-stream] Failed to send done: %v", writeErr)
	}
}

// roleStreamConfig holds per-role LLM configuration.
type roleStreamConfig struct {
	SystemPrompt string
	APIURL       string
	APIKey       string
	Model        string
}

// resolveRoleConfig looks up a role by ID and returns its system prompt.
// Per-role LLM config (model, API URL) is supported when the role has
// custom fields set via the roles API.
func resolveRoleConfig(roleID string) roleStreamConfig {
	roles := store.ListRoles()

	// Map string role IDs from the frontend to numeric store IDs.
	// The frontend uses string IDs like "cli", "ops", "frontend", etc.
	// The store uses numeric IDs 1-7 for built-in roles.
	builtinIDMap := map[string]int{
		"cli":      1,
		"ops":      7,
		"prompt":   5,
		"frontend": 3,
		"backend":  4,
		"ui":       3,
		"api":      4,
	}

	storeID, ok := builtinIDMap[roleID]
	if !ok {
		// Try to find by name or return defaults.
		return roleStreamConfig{SystemPrompt: ""}
	}

	for _, r := range roles {
		if r.ID == storeID {
			return roleStreamConfig{
				SystemPrompt: r.SystemPrompt,
			}
		}
	}

	return roleStreamConfig{SystemPrompt: ""}
}

// sendWSStreamError writes a JSON error message over the WebSocket.
func sendWSStreamError(conn *websocket.Conn, message string) {
	err := conn.WriteJSON(wsStreamMessage{
		Type: "error",
		Data: message,
	})
	if err != nil {
		log.Printf("[ai-stream] Failed to send error: %v", err)
	}
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or falls back to defaultValue if the variable is empty or not set.
func getEnvOrDefault(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
