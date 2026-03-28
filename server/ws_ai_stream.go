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
	Type string `json:"type"` // "token", "done", or "error"
	Data string `json:"data"`
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

	systemPrompt := resolveSystemPrompt(req.Role)
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant that generates terminal commands. Respond concisely."
	}

	apiURL := getEnvOrDefault("OPENAI_API_URL", "https://api.openai.com/v1")
	model := getEnvOrDefault("OPENAI_MODEL", "gpt-4")
	apiKey := os.Getenv("OPENAI_API_KEY")

	var fullText string
	err = ai.StreamChatCompletion(
		r.Context(),
		apiURL,
		apiKey,
		model,
		systemPrompt,
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

	if writeErr := conn.WriteJSON(wsStreamMessage{
		Type: "done",
		Data: fullText,
	}); writeErr != nil {
		log.Printf("[ai-stream] Failed to send done: %v", writeErr)
	}
}

// builtinRoleIDMap maps frontend role string IDs to store numeric IDs.
var builtinRoleIDMap = map[string]int{
	"cli": 1,
	"ops": 7,
	"prompt":   5,
	"frontend": 3,
	"backend":  4,
	"ui":       3,
	"api":      4,
}

// resolveSystemPrompt looks up a role by its frontend string ID and returns
// its system prompt. Returns an empty string if the role is not found.
func resolveSystemPrompt(roleID string) string {
	storeID, ok := builtinRoleIDMap[roleID]
	if !ok {
		return ""
	}

	for _, r := range store.ListRoles() {
		if r.ID == storeID {
			return r.SystemPrompt
		}
	}

	return ""
}

// sendWSStreamError writes a JSON error message over the WebSocket.
func sendWSStreamError(conn *websocket.Conn, message string) {
	if err := conn.WriteJSON(wsStreamMessage{
		Type: "error",
		Data: message,
	}); err != nil {
		log.Printf("[ai-stream] Failed to send error: %v", err)
	}
}

// getEnvOrDefault returns the value of the environment variable named by key,
// or defaultValue if the variable is empty or not set.
func getEnvOrDefault(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
