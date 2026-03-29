package server

import (
	"sync"
	"testing"

	"ttyweb/db"
	"ttyweb/service"
)

// testDB initializes a temporary SQLite database and returns a cleanup function.
// It also resets the global project service singleton so tests get a fresh instance.
func testDB(t *testing.T) {
	t.Helper()
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
}

// newTestServerWithDB creates a test server with DB-backed services.
// It resets the global MemoryStore and project service singleton to ensure test isolation.
func newTestServerWithDB(t *testing.T) *Server {
	t.Helper()
	store = NewMemoryStore()
	testDB(t)

	// Reset the projectService singleton so it re-initializes with the new DB.
	projectServiceOnce = sync.Once{}
	projectServiceInstance = nil
	errProjectService = nil

	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}

	return &Server{
		options:        &Options{Path: "/"},
		noteSvc:        service.NewNoteService(gormDB),
		segmentService: service.NewTaskSegmentService(gormDB),
	}
}
