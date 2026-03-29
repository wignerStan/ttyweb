package ai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// xfyunHost is the Xunfei IAT WebSocket host.
	xfyunHost = "iat.xf-yun.com"
	// xfyunPath is the Xunfei IAT WebSocket path.
	xfyunPath = "/v1"
)

// XunfeiConfig holds credentials for the Xunfei IAT STT service.
type XunfeiConfig struct {
	AppID     string
	APIKey    string
	APISecret string
}

// IsConfigured returns true if all required fields are set.
func (c XunfeiConfig) IsConfigured() bool {
	return c.AppID != "" && c.APIKey != "" && c.APISecret != ""
}

// SpeechParams holds parameters for speech recognition.
type SpeechParams struct {
	Language string // e.g. "zh_cn" (default)
	Domain   string // e.g. "iat" or "slm"
	Accent   string // e.g. "mandarin"
	Hotwords string // pipe-delimited hotwords string
}

// DefaultSpeechParams returns sensible defaults for Chinese Mandarin IAT.
func DefaultSpeechParams() SpeechParams {
	return SpeechParams{
		Language: "zh_cn",
		Domain:   "iat",
		Accent:   "mandarin",
	}
}

// SpeechResult holds a parsed result from Xunfei IAT.
type SpeechResult struct {
	Text       string
	IsComplete bool
	Status     int // 0=first, 1=middle, 2=last
	SN         int
	LS         bool
	PGS        string // "apd"=append, "rpl"=replace
	RG         [2]int // replace range [from, to]
}

// GenerateAuthURL builds the HMAC-SHA256 signed WebSocket URL for Xunfei IAT.
// It follows the official Xunfei IAT v2 authentication scheme:
//   - signatureOrigin = "host: {host}\ndate: {date}\nGET {path} HTTP/1.1"
//   - signature = HMAC-SHA256(apiSecret, signatureOrigin) encoded as base64
//   - authorization = base64("api_key=\"...\", algorithm=\"hmac-sha256\", ...")
func GenerateAuthURL(config XunfeiConfig) (string, error) {
	if !config.IsConfigured() {
		return "", fmt.Errorf("xunfei config is incomplete: appID, apiKey, and apiSecret are required")
	}

	date := time.Now().UTC().Format(http.TimeFormat)

	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", xfyunHost, date, xfyunPath)

	mac := hmac.New(sha256.New, []byte(config.APISecret))
	mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	authOrigin := fmt.Sprintf(
		`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		config.APIKey, signature,
	)
	authorization := base64.StdEncoding.EncodeToString([]byte(authOrigin))

	u := url.URL{
		Scheme: "wss",
		Host:   xfyunHost,
		Path:   xfyunPath,
		RawQuery: fmt.Sprintf("authorization=%s&date=%s&host=%s",
			url.QueryEscape(authorization),
			url.QueryEscape(date),
			url.QueryEscape(xfyunHost),
		),
	}
	return u.String(), nil
}

// xfyunFrame is the JSON frame structure for Xunfei IAT v2 WebSocket messages.
type xfyunFrame struct {
	Header    xfyunFrameHeader  `json:"header"`
	Parameter xfyunFrameParams  `json:"parameter"`
	Payload   xfyunFramePayload `json:"payload"`
}

// xfyunFrameHeader is the header of a Xunfei IAT v2 frame.
type xfyunFrameHeader struct {
	AppID  string `json:"app_id"`
	Status int    `json:"status"`
}

// xfyunFrameParams holds IAT-specific parameters.
type xfyunFrameParams struct {
	IAT map[string]interface{} `json:"iat"`
}

// xfyunFramePayload wraps the audio payload.
type xfyunFramePayload struct {
	Audio xfyunAudioPayload `json:"audio"`
}

// xfyunAudioPayload describes the audio encoding and data.
type xfyunAudioPayload struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	BitDepth   int    `json:"bit_depth"`
	Seq        int    `json:"seq"`
	Status     int    `json:"status"`
	Audio      string `json:"audio"`
}

// BuildFirstFrame builds the first WebSocket frame with IAT parameters.
// The audio parameter is the base64-encoded PCM audio data (may be empty).
// seq is the sequence number starting from 1.
func BuildFirstFrame(config XunfeiConfig, params SpeechParams, audio string, seq int) ([]byte, error) {
	if !config.IsConfigured() {
		return nil, fmt.Errorf("xunfei config is incomplete")
	}

	iatParams := map[string]interface{}{
		"domain":   params.Domain,
		"language": params.Language,
		"accent":   params.Accent,
		"eos":      6000,
		"vinfo":    1,
		"dwa":      "wpgs",
		"result": map[string]string{
			"encoding": "utf8",
			"compress": "raw",
			"format":   "json",
		},
	}
	if params.Hotwords != "" {
		iatParams["dhw"] = "utf-8;" + params.Hotwords
	}

	frame := xfyunFrame{
		Header:    xfyunFrameHeader{AppID: config.AppID, Status: 0},
		Parameter: xfyunFrameParams{IAT: iatParams},
		Payload: xfyunFramePayload{
			Audio: xfyunAudioPayload{
				Encoding:   "raw",
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
				Seq:        seq,
				Status:     0,
				Audio:      audio,
			},
		},
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("BuildFirstFrame: marshal frame: %w", err)
	}
	return data, nil
}

// BuildMiddleFrame builds a middle (continuation) WebSocket frame with audio data.
// audio is base64-encoded PCM audio. seq is the sequence number.
func BuildMiddleFrame(audio string, seq int) ([]byte, error) {
	frame := xfyunFrame{
		Header: xfyunFrameHeader{Status: 1},
		Payload: xfyunFramePayload{
			Audio: xfyunAudioPayload{
				Encoding:   "raw",
				SampleRate: 16000,
				Seq:        seq,
				Status:     1,
				Audio:      audio,
			},
		},
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("BuildMiddleFrame: marshal frame: %w", err)
	}
	return data, nil
}

// BuildLastFrame builds the final WebSocket frame signalling end of audio.
// seq is the final sequence number.
func BuildLastFrame(seq int) ([]byte, error) {
	frame := xfyunFrame{
		Header: xfyunFrameHeader{Status: 2},
		Payload: xfyunFramePayload{
			Audio: xfyunAudioPayload{
				Encoding:   "raw",
				SampleRate: 16000,
				Seq:        seq,
				Status:     2,
			},
		},
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("BuildLastFrame: marshal frame: %w", err)
	}
	return data, nil
}

// xfyunResponse represents the top-level JSON response from Xunfei IAT v2.
type xfyunResponse struct {
	Header  xfyunResponseHeader   `json:"header"`
	Payload *xfyunResponsePayload `json:"payload,omitempty"`
}

// xfyunResponseHeader is the header of a Xunfei IAT v2 response.
type xfyunResponseHeader struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// xfyunResponsePayload wraps the optional result payload.
type xfyunResponsePayload struct {
	Result *xfyunResultPayload `json:"result,omitempty"`
}

// xfyunResultPayload holds the base64-encoded recognition result.
type xfyunResultPayload struct {
	Text string `json:"text"` // base64-encoded JSON
}

// xfyunTextData is the decoded inner JSON from the result text field.
type xfyunTextData struct {
	SN  int         `json:"sn"`
	LS  bool        `json:"ls"`
	PGS string      `json:"pgs"`
	RG  []int       `json:"rg"`
	WS  []xfyunWord `json:"ws"`
}

// xfyunWord is a word slot in the recognition result.
type xfyunWord struct {
	CW []xfyunCharWord `json:"cw"`
}

// xfyunCharWord is a character word within a word slot.
type xfyunCharWord struct {
	W string `json:"w"`
}

// ParseResult parses a Xunfei JSON response and extracts text.
// Returns an error if the response indicates a server-side error.
func ParseResult(data []byte) (*SpeechResult, error) {
	var resp xfyunResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse xunfei response: %w", err)
	}

	if resp.Header.Code != 0 {
		return nil, fmt.Errorf("xunfei error (code %d): %s", resp.Header.Code, resp.Header.Message)
	}

	result := &SpeechResult{
		Status: resp.Header.Status,
	}

	if resp.Payload == nil || resp.Payload.Result == nil || resp.Payload.Result.Text == "" {
		return result, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Payload.Result.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result text: %w", err)
	}

	var textData xfyunTextData
	if err := json.Unmarshal(decoded, &textData); err != nil {
		return nil, fmt.Errorf("failed to parse decoded text: %w", err)
	}

	var text strings.Builder
	for _, word := range textData.WS {
		for _, cw := range word.CW {
			text.WriteString(cw.W)
		}
	}

	result.Text = text.String()
	result.SN = textData.SN
	result.LS = textData.LS
	result.PGS = textData.PGS
	result.IsComplete = resp.Header.Status == 2

	if len(textData.RG) == 2 {
		result.RG = [2]int{textData.RG[0], textData.RG[1]}
	}

	return result, nil
}

// LoadXunfeiConfigFromEnv loads Xunfei credentials from environment variables:
// XFYUN_APP_ID, XFYUN_API_KEY, XFYUN_API_SECRET.
func LoadXunfeiConfigFromEnv() XunfeiConfig {
	return XunfeiConfig{
		AppID:     os.Getenv("XFYUN_APP_ID"),
		APIKey:    os.Getenv("XFYUN_API_KEY"),
		APISecret: os.Getenv("XFYUN_API_SECRET"),
	}
}
