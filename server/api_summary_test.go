package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/ai"
	"ttyweb/db"
	"ttyweb/service"
)

// newTestServerWithSummaryService creates a test server with a SummaryService
// backed by the test DB and a mock AI server.
func newTestServerWithSummaryService(t *testing.T, mockAISrvURL string) *Server {
	t.Helper()
	store = NewMemoryStore()
	testDB(t)

	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}

	aiClient := ai.NewClient("test-key", mockAISrvURL, "test-model")
	summarySvc := service.NewSummaryService(gormDB, aiClient)

	return &Server{
		options:        &Options{Path: "/"},
		noteSvc:        service.NewNoteService(gormDB),
		segmentService: service.NewTaskSegmentService(gormDB),
		summaryService: summarySvc,
	}
}

func TestHandleSummarizeSegment_MethodNotAllowed(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()

	srv := newTestServerWithSummaryService(t, mockSrv.URL)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/seg-001/summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSummarizeSegment_MissingID(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()

	srv := newTestServerWithSummaryService(t, mockSrv.URL)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments//summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetSummary_NotFound(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()

	srv := newTestServerWithSummaryService(t, mockSrv.URL)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent/summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListSummaries_Empty(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()

	srv := newTestServerWithSummaryService(t, mockSrv.URL)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool             `json:"success"`
		Data    []db.TaskSummary `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Data == nil {
		t.Error("expected non-nil data array")
	}
}

// newMockAIServerForHandlerTest creates a minimal mock AI server for handler tests.
func newMockAIServerForHandlerTest(t *testing.T, cannedResponse string, statusCode int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != 0 {
			w.WriteHeader(statusCode)
			return
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": cannedResponse}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}
