package ai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

const (
	testAppID     = "test123"
	testAPIKey    = "key456"
	testAPISecret = "secret789"
)

func testConfig() XunfeiConfig {
	return XunfeiConfig{
		AppID:     testAppID,
		APIKey:    testAPIKey,
		APISecret: testAPISecret,
	}
}

// --- GenerateAuthURL ---

func TestGenerateAuthURL_EmptyConfig(t *testing.T) {
	_, err := GenerateAuthURL(XunfeiConfig{})
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestGenerateAuthURL_PartialConfig(t *testing.T) {
	_, err := GenerateAuthURL(XunfeiConfig{AppID: "a"})
	if err == nil {
		t.Fatal("expected error for partial config")
	}
}

func TestGenerateAuthURL_ValidURL(t *testing.T) {
	cfg := testConfig()
	authURL, err := GenerateAuthURL(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	if u.Scheme != "wss" {
		t.Errorf("expected scheme wss, got %s", u.Scheme)
	}
	if u.Host != xfyunHost {
		t.Errorf("expected host %s, got %s", xfyunHost, u.Host)
	}
	if u.Path != xfyunPath {
		t.Errorf("expected path %s, got %s", xfyunPath, u.Path)
	}

	// Verify required query params exist.
	q := u.Query()
	requiredParams := []string{"authorization", "date", "host"}
	for _, p := range requiredParams {
		if q.Get(p) == "" {
			t.Errorf("missing required query param: %s", p)
		}
	}

	// Verify the authorization can be decoded and contains the API key.
	authDecoded, err := base64.StdEncoding.DecodeString(q.Get("authorization"))
	if err != nil {
		t.Fatalf("failed to decode authorization: %v", err)
	}
	authStr := string(authDecoded)
	if !strings.Contains(authStr, testAPIKey) {
		t.Errorf("authorization does not contain API key")
	}
	if !strings.Contains(authStr, "hmac-sha256") {
		t.Errorf("authorization does not contain algorithm")
	}
}

func TestGenerateAuthURL_SignatureVerifiable(t *testing.T) {
	cfg := testConfig()
	authURL, err := GenerateAuthURL(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	q := u.Query()

	// Decode authorization to extract the signature.
	authDecoded, err := base64.StdEncoding.DecodeString(q.Get("authorization"))
	if err != nil {
		t.Fatalf("failed to decode authorization: %v", err)
	}

	// Reconstruct the signature origin and verify the HMAC matches.
	date := q.Get("date")
	signatureOrigin := "host: " + xfyunHost + "\ndate: " + date + "\nGET " + xfyunPath + " HTTP/1.1"

	mac := hmac.New(sha256.New, []byte(testAPISecret))
	mac.Write([]byte(signatureOrigin))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	authStr := string(authDecoded)
	if !strings.Contains(authStr, expectedSig) {
		t.Errorf("signature mismatch: expected %s to be in %s", expectedSig, authStr)
	}
}

// --- BuildFirstFrame ---

func TestBuildFirstFrame_EmptyConfig(t *testing.T) {
	_, err := BuildFirstFrame(XunfeiConfig{}, DefaultSpeechParams(), "", 1)
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestBuildFirstFrame_Valid(t *testing.T) {
	cfg := testConfig()
	params := DefaultSpeechParams()

	frame, err := BuildFirstFrame(cfg, params, "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("frame is not valid JSON: %v", err)
	}

	// Check header status = 0
	header := parsed["header"].(map[string]interface{})
	if int(header["status"].(float64)) != 0 {
		t.Errorf("expected header status 0, got %v", header["status"])
	}
	if header["app_id"] != testAppID {
		t.Errorf("expected app_id %s, got %v", testAppID, header["app_id"])
	}

	// Check payload audio status = 0
	payload := parsed["payload"].(map[string]interface{})
	audio := payload["audio"].(map[string]interface{})
	if int(audio["status"].(float64)) != 0 {
		t.Errorf("expected audio status 0, got %v", audio["status"])
	}

	// Check IAT parameters exist
	param := parsed["parameter"].(map[string]interface{})
	iat := param["iat"].(map[string]interface{})
	if iat["language"] != "zh_cn" {
		t.Errorf("expected language zh_cn, got %v", iat["language"])
	}
	if iat["domain"] != "iat" {
		t.Errorf("expected domain iat, got %v", iat["domain"])
	}
}

func TestBuildFirstFrame_WithHotwords(t *testing.T) {
	cfg := testConfig()
	params := DefaultSpeechParams()
	params.Hotwords = "hello|world"

	frame, err := BuildFirstFrame(cfg, params, "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("frame is not valid JSON: %v", err)
	}

	param := parsed["parameter"].(map[string]interface{})
	iat := param["iat"].(map[string]interface{})
	if iat["dhw"] != "utf-8;hello|world" {
		t.Errorf("expected dhw, got %v", iat["dhw"])
	}
}

func TestBuildFirstFrame_WithAudio(t *testing.T) {
	cfg := testConfig()
	params := DefaultSpeechParams()
	audioB64 := base64.StdEncoding.EncodeToString([]byte("pcmdata"))

	frame, err := BuildFirstFrame(cfg, params, audioB64, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("frame is not valid JSON: %v", err)
	}

	payload := parsed["payload"].(map[string]interface{})
	audio := payload["audio"].(map[string]interface{})
	if audio["audio"] != audioB64 {
		t.Errorf("audio data mismatch")
	}
}

// --- BuildMiddleFrame ---

func TestBuildMiddleFrame(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte("morepcm"))

	frame, err := BuildMiddleFrame(audioB64, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("frame is not valid JSON: %v", err)
	}

	header := parsed["header"].(map[string]interface{})
	if int(header["status"].(float64)) != 1 {
		t.Errorf("expected header status 1, got %v", header["status"])
	}

	payload := parsed["payload"].(map[string]interface{})
	audio := payload["audio"].(map[string]interface{})
	if int(audio["status"].(float64)) != 1 {
		t.Errorf("expected audio status 1, got %v", audio["status"])
	}
	if audio["audio"] != audioB64 {
		t.Errorf("audio data mismatch")
	}
}

// --- BuildLastFrame ---

func TestBuildLastFrame(t *testing.T) {
	frame, err := BuildLastFrame(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(frame, &parsed); err != nil {
		t.Fatalf("frame is not valid JSON: %v", err)
	}

	header := parsed["header"].(map[string]interface{})
	if int(header["status"].(float64)) != 2 {
		t.Errorf("expected header status 2, got %v", header["status"])
	}

	payload := parsed["payload"].(map[string]interface{})
	audio := payload["audio"].(map[string]interface{})
	if int(audio["status"].(float64)) != 2 {
		t.Errorf("expected audio status 2, got %v", audio["status"])
	}
	if audio["audio"] != "" {
		t.Errorf("expected empty audio in last frame, got %v", audio["audio"])
	}
}

// --- ParseResult ---

func TestParseResult_ErrorResponse(t *testing.T) {
	resp := `{"header":{"code":10105,"message":"illegal access","status":0}}`
	_, err := ParseResult([]byte(resp))
	if err == nil {
		t.Fatal("expected error for error response")
	}
}

func TestParseResult_EmptyResult(t *testing.T) {
	resp := `{"header":{"code":0,"message":"success","status":0}}`
	result, err := ParseResult([]byte(resp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "" {
		t.Errorf("expected empty text, got %q", result.Text)
	}
	if result.IsComplete {
		t.Error("expected not complete")
	}
}

func TestParseResult_PartialResult(t *testing.T) {
	// Build a realistic Xunfei response with base64-encoded text payload.
	innerText := `{"sn":1,"ls":false,"bg":0,"ed":0,"lsd":false,"pgs":"apd","rg":[0,100],"ws":[{"cw":[{"w":"你"},{"w":"好"}]}]}`
	encodedText := base64.StdEncoding.EncodeToString([]byte(innerText))

	resp := `{"header":{"code":0,"message":"success","status":1},"payload":{"result":{"text":"` + encodedText + `"}}}`

	result, err := ParseResult([]byte(resp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "你好" {
		t.Errorf("expected text '你好', got %q", result.Text)
	}
	if result.SN != 1 {
		t.Errorf("expected SN 1, got %d", result.SN)
	}
	if result.PGS != "apd" {
		t.Errorf("expected PGS apd, got %s", result.PGS)
	}
	if result.IsComplete {
		t.Error("expected not complete")
	}
}

func TestParseResult_FinalResult(t *testing.T) {
	innerText := `{"sn":2,"ls":true,"bg":0,"ed":0,"lsd":true,"pgs":"apd","rg":[0,100],"ws":[{"cw":[{"w":"世"},{"w":"界"}]}]}`
	encodedText := base64.StdEncoding.EncodeToString([]byte(innerText))

	resp := `{"header":{"code":0,"message":"success","status":2},"payload":{"result":{"text":"` + encodedText + `"}}}`

	result, err := ParseResult([]byte(resp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "世界" {
		t.Errorf("expected text '世界', got %q", result.Text)
	}
	if !result.IsComplete {
		t.Error("expected complete")
	}
	if result.Status != 2 {
		t.Errorf("expected status 2, got %d", result.Status)
	}
	if !result.LS {
		t.Error("expected LS true")
	}
}

func TestParseResult_ReplaceResult(t *testing.T) {
	innerText := `{"sn":5,"ls":false,"bg":0,"ed":0,"lsd":false,"pgs":"rpl","rg":[3,5],"ws":[{"cw":[{"w":"修"},{"w":"改"}]}]}`
	encodedText := base64.StdEncoding.EncodeToString([]byte(innerText))

	resp := `{"header":{"code":0,"message":"success","status":1},"payload":{"result":{"text":"` + encodedText + `"}}}`

	result, err := ParseResult([]byte(resp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PGS != "rpl" {
		t.Errorf("expected PGS rpl, got %s", result.PGS)
	}
	if result.RG[0] != 3 || result.RG[1] != 5 {
		t.Errorf("expected RG [3,5], got %v", result.RG)
	}
}

func TestParseResult_InvalidJSON(t *testing.T) {
	_, err := ParseResult([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseResult_NoPayload(t *testing.T) {
	resp := `{"header":{"code":0,"message":"success","status":0}}`
	result, err := ParseResult([]byte(resp))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "" {
		t.Errorf("expected empty text, got %q", result.Text)
	}
}

func TestParseResult_InvalidBase64(t *testing.T) {
	resp := `{"header":{"code":0,"message":"success","status":1},"payload":{"result":{"text":"not-valid-base64!!!"}}}`
	_, err := ParseResult([]byte(resp))
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestParseResult_ValidBase64InvalidJSON(t *testing.T) {
	encodedText := base64.StdEncoding.EncodeToString([]byte("not json at all"))
	resp := `{"header":{"code":0,"message":"success","status":1},"payload":{"result":{"text":"` + encodedText + `"}}}`
	_, err := ParseResult([]byte(resp))
	if err == nil {
		t.Fatal("expected error for invalid JSON after base64 decode, got nil")
	}
}

func TestLoadXunfeiConfigFromEnv(t *testing.T) {
	t.Setenv("XFYUN_APP_ID", "test-app")
	t.Setenv("XFYUN_API_KEY", "test-key")
	t.Setenv("XFYUN_API_SECRET", "test-secret")

	cfg := LoadXunfeiConfigFromEnv()
	if cfg.AppID != "test-app" {
		t.Errorf("AppID = %q, want %q", cfg.AppID, "test-app")
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.APISecret != "test-secret" {
		t.Errorf("APISecret = %q, want %q", cfg.APISecret, "test-secret")
	}
}

// --- IsConfigured ---

func TestXunfeiConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  XunfeiConfig
		want bool
	}{
		{"all set", testConfig(), true},
		{"missing appID", XunfeiConfig{APIKey: "a", APISecret: "b"}, false},
		{"missing apiKey", XunfeiConfig{AppID: "a", APISecret: "b"}, false},
		{"missing secret", XunfeiConfig{AppID: "a", APIKey: "b"}, false},
		{"all empty", XunfeiConfig{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- DefaultSpeechParams ---

func TestDefaultSpeechParams(t *testing.T) {
	p := DefaultSpeechParams()
	if p.Language != "zh_cn" {
		t.Errorf("expected language zh_cn, got %s", p.Language)
	}
	if p.Domain != "iat" {
		t.Errorf("expected domain iat, got %s", p.Domain)
	}
	if p.Accent != "mandarin" {
		t.Errorf("expected accent mandarin, got %s", p.Accent)
	}
}
