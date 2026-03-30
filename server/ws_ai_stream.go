package server

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"

	"ttyweb/ai"
	"ttyweb/pkg/validate"
)

// wsStreamMessage is a JSON message sent over the AI stream WebSocket.
type wsStreamMessage struct {
	Type string `json:"type"` // "token", "done", or "error"
	Data string `json:"data"`
}

// builtinRoleIDMap maps frontend role IDs to store IDs.
var builtinRoleIDMap = map[string]int{
	"cli":      1,
	"ops":      7,
	"prompt":   5,
	"frontend": 3,
	"backend":  4,
	"ui":       3,
	"api":      4,
}

const defaultSystemPrompt = "You are a helpful assistant that generates terminal commands. Respond concisely."

// readTimeout is the maximum time to wait for the initial WebSocket message.
const readTimeout = 10 * time.Second

// handleAIStream handles the /ws/ai/stream WebSocket endpoint.
// It reads a JSON message with role and prompt fields, then streams
// LLM responses back as individual token messages followed by a done message.
func (server *Server) handleAIStream(w http.ResponseWriter, r *http.Request) {
	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ai-stream] WebSocket upgrade failed: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Set a read deadline so connections that never send data are cleaned up.
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("[ai-stream] Failed to set read deadline: %v", err)
		return
	}

	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[ai-stream] Failed to read initial message: %v", err)
		return
	}

	// Clear the read deadline after the initial message is received.
	_ = conn.SetReadDeadline(time.Time{})

	var req struct {
		Role      string `json:"role"`
		Prompt    string `json:"prompt"`
		AuthToken string `json:"auth_token"`
	}
	if err := json.Unmarshal(msgBytes, &req); err != nil {
		sendWSStreamError(conn, "invalid request format")
		log.Printf("[ai-stream] JSON decode failed: %v", err)
		return
	}

	// Verify authentication credential using constant-time comparison,
	// matching the pattern in processWSConn (handlers.go).
	cred := []byte(server.options.Credential)
	if subtle.ConstantTimeCompare([]byte(req.AuthToken), cred) != 1 {
		sendWSStreamError(conn, "authentication failed")
		log.Printf("[ai-stream] Authentication failed from %s", r.RemoteAddr)
		return
	}

	if req.Prompt == "" {
		sendWSStreamError(conn, "prompt is required")
		return
	}

	// Validate the API URL to prevent SSRF attacks.
	apiURL := envOrDefault("OPENAI_API_URL", "https://api.openai.com/v1")
	if err := validate.APIURL(apiURL); err != nil {
		log.Printf("[ai-stream] API URL validation failed for %q: %v", apiURL, err)
		sendWSStreamError(conn, "AI service unavailable")
		return
	}

	systemPrompt := resolveSystemPrompt(req.Role)
	model := envOrDefault("OPENAI_MODEL", "gpt-4")

	var fullText string
	err = ai.StreamChatCompletion(
		r.Context(),
		apiURL,
		os.Getenv("OPENAI_API_KEY"),
		model,
		systemPrompt,
		req.Prompt,
		func(token string) {
			fullText += token
			if writeErr := conn.WriteJSON(wsStreamMessage{Type: "token", Data: token}); writeErr != nil {
				log.Printf("[ai-stream] Failed to send token: %v", writeErr)
			}
		},
	)

	if err != nil {
		log.Printf("[ai-stream] Stream completion failed: %v", err)
		sendWSStreamError(conn, sanitizeStreamError(err))
		return
	}

	if writeErr := conn.WriteJSON(wsStreamMessage{Type: "done", Data: fullText}); writeErr != nil {
		log.Printf("[ai-stream] Failed to send done: %v", writeErr)
	}
}

// sanitizeStreamError returns a generic user-safe error message.
// The full error is already logged server-side.
func sanitizeStreamError(err error) string {
	_ = err // logged at call site
	return "AI stream error"
}

// resolveSystemPrompt looks up the system prompt for the given frontend role ID.
func resolveSystemPrompt(roleID string) string {
	storeID, ok := builtinRoleIDMap[roleID]
	if !ok {
		return defaultSystemPrompt
	}

	for _, r := range store.ListRoles() {
		if r.ID == storeID {
			return r.SystemPrompt
		}
	}
	return defaultSystemPrompt
}

// sendWSStreamError writes a JSON error message over the WebSocket.
func sendWSStreamError(conn *websocket.Conn, message string) {
	if err := conn.WriteJSON(wsStreamMessage{Type: "error", Data: message}); err != nil {
		log.Printf("[ai-stream] Failed to send error: %v", err)
	}
}

// envOrDefault returns the value of the environment variable named by key,
// or falls back to defaultValue if the variable is empty or not set.
func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
