package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"ttyweb/ai"
)

// speechUpgrader is the WebSocket upgrader for the speech proxy endpoint.
var speechUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// clientMessage is a message received from the browser client.
type clientMessage struct {
	Type  string `json:"type"`
	Audio string `json:"audio,omitempty"`
}

// serverMessage is a message sent to the browser client.
type serverMessage struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	IsComplete bool   `json:"isComplete,omitempty"`
	Message    string `json:"message,omitempty"`
	SN         int    `json:"sn,omitempty"`
	LS         bool   `json:"ls,omitempty"`
	PGS        string `json:"pgs,omitempty"`
	RG         []int  `json:"rg,omitempty"`
}

// speechSession holds shared state for a single speech proxy session,
// coordinating the client and Xunfei WebSocket connections.
type speechSession struct {
	clientConn *websocket.Conn
	xunfeiConn *websocket.Conn
	xunfeiCfg  ai.XunfeiConfig
	params     ai.SpeechParams
	seq        int
	mu         sync.Mutex
	done       chan struct{}
}

func (s *speechSession) closeBoth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.done:
		return
	default:
	}
	close(s.done)
	if s.xunfeiConn != nil {
		_ = s.xunfeiConn.Close()
		s.xunfeiConn = nil
	}
	if s.clientConn != nil {
		_ = s.clientConn.Close()
		s.clientConn = nil
	}
}

func (s *speechSession) sendError(msg string) {
	_ = s.clientConn.WriteJSON(serverMessage{Type: "error", Message: msg})
}

// handleSpeechWS handles WebSocket connections to /ws/speech.
// It proxies audio between the browser and the Xunfei IAT STT service.
func (_ *Server) handleSpeechWS(w http.ResponseWriter, r *http.Request) {
	xunfeiCfg := ai.LoadXunfeiConfigFromEnv()

	if !xunfeiCfg.IsConfigured() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(serverMessage{
			Type:    "error",
			Message: "Xunfei STT is not configured. Set XFYUN_APP_ID, XFYUN_API_KEY, and XFYUN_API_SECRET environment variables.",
		})
		return
	}

	conn, err := speechUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Speech] Failed to upgrade WebSocket: %v", err)
		return
	}
	defer func() { _ = conn.Close() }()

	log.Printf("[Speech] Client connected: %s", r.RemoteAddr)

	sess := &speechSession{
		clientConn: conn,
		xunfeiCfg:  xunfeiCfg,
		params:     ai.DefaultSpeechParams(),
		done:       make(chan struct{}),
	}
	sess.run()
}

// run processes client messages and proxies audio to Xunfei.
func (s *speechSession) run() {
	for {
		_, raw, err := s.clientConn.ReadMessage()
		if err != nil {
			log.Printf("[Speech] Client read error: %v", err)
			s.closeBoth()
			return
		}

		var msg clientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("[Speech] Invalid client message: %v", err)
			continue
		}

		switch msg.Type {
		case "start":
			s.handleStart()

		case "audio":
			if s.xunfeiConn == nil {
				continue
			}
			s.seq++
			frame, err := s.buildAudioFrame(msg.Audio)
			if err != nil {
				s.sendError("failed to build audio frame: " + err.Error())
				s.closeBoth()
				return
			}
			if err := s.xunfeiConn.WriteMessage(websocket.TextMessage, frame); err != nil {
				log.Printf("[Speech] Error sending to Xunfei: %v", err)
				s.closeBoth()
				return
			}

		case "stop":
			if s.xunfeiConn == nil {
				continue
			}
			s.seq++
			frame, err := ai.BuildLastFrame(s.seq)
			if err != nil {
				s.sendError("failed to build last frame: " + err.Error())
				s.closeBoth()
				return
			}
			if err := s.xunfeiConn.WriteMessage(websocket.TextMessage, frame); err != nil {
				log.Printf("[Speech] Error sending last frame: %v", err)
			}

		default:
			log.Printf("[Speech] Unknown message type: %s", msg.Type)
		}
	}
}

func (s *speechSession) buildAudioFrame(audio string) ([]byte, error) {
	if s.seq == 1 {
		frame, err := ai.BuildFirstFrame(s.xunfeiCfg, s.params, audio, s.seq)
		if err != nil {
			return nil, errors.Wrapf(err, "buildAudioFrame: first frame")
		}
		return frame, nil
	}
	frame, err := ai.BuildMiddleFrame(audio, s.seq)
	if err != nil {
		return nil, errors.Wrapf(err, "buildAudioFrame: middle frame")
	}
	return frame, nil
}

// handleStart connects to Xunfei and starts the result relay goroutine.
func (s *speechSession) handleStart() {
	authURL, err := ai.GenerateAuthURL(s.xunfeiCfg)
	if err != nil {
		s.sendError("failed to generate auth URL: " + err.Error())
		s.closeBoth()
		return
	}

	log.Printf("[Speech] Connecting to Xunfei...")
	xfConn, xfResp, err := websocket.DefaultDialer.Dial(authURL, nil)
	if err != nil {
		s.sendError("failed to connect to Xunfei: " + err.Error())
		s.closeBoth()
		return
	}
	if xfResp != nil {
		_ = xfResp.Body.Close()
	}

	s.mu.Lock()
	s.xunfeiConn = xfConn
	s.seq = 0
	s.mu.Unlock()

	log.Printf("[Speech] Xunfei connected")

	if err := s.clientConn.WriteJSON(serverMessage{Type: "ready"}); err != nil {
		log.Printf("[Speech] Error sending ready: %v", err)
		s.closeBoth()
		return
	}

	go s.relayXunfeiToClient(xfConn)
}

// relayXunfeiToClient reads messages from Xunfei and forwards parsed results to the client.
func (s *speechSession) relayXunfeiToClient(xfConn *websocket.Conn) {
	defer func() { _ = xfConn.Close() }()

	for {
		_, raw, err := xfConn.ReadMessage()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Printf("[Speech] Xunfei read error: %v", err)
				s.sendError("Xunfei connection error: " + err.Error())
			}
			return
		}

		result, err := ai.ParseResult(raw)
		if err != nil {
			log.Printf("[Speech] Parse error: %v", err)
			s.sendError(err.Error())
			continue
		}

		if result.Text != "" {
			out := serverMessage{
				Type: "partial",
				Text: result.Text,
				SN:   result.SN,
				LS:   result.LS,
				PGS:  result.PGS,
			}
			if result.RG[0] != 0 || result.RG[1] != 0 {
				out.RG = []int{result.RG[0], result.RG[1]}
			}
			if err := s.clientConn.WriteJSON(out); err != nil {
				log.Printf("[Speech] Error sending result to client: %v", err)
				return
			}
		}

		if result.IsComplete {
			log.Printf("[Speech] Recognition complete")
			if err := s.clientConn.WriteJSON(serverMessage{Type: "end"}); err != nil {
				log.Printf("[Speech] Error sending end: %v", err)
			}
			return
		}
	}
}
