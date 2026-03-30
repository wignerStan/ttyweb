package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdates_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, err := CheckForUpdates(ctx, "1.0.0")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestCheckForUpdates_InvalidRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Not Found",
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := checkForUpdatesURL(ctx, "1.0.0", srv.URL+"/repos/invalid/nonexistent/releases/latest")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.Equal(t, "", result.LatestVersion)
	assert.False(t, result.HasUpdate)
	assert.Greater(t, result.CheckedAt, int64(0))
}

func TestCheckForUpdates_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "internal server error"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := checkForUpdatesURL(ctx, "1.0.0", srv.URL+"/repos/test/test/releases/latest")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")
}

func TestCheckForUpdates_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v2.0.0",
			"html_url": "https://github.com/example/repo/releases/tag/v2.0.0",
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := checkForUpdatesURL(ctx, "1.0.0", srv.URL+"/repos/example/repo/releases/latest")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1.0.0", result.CurrentVersion)
	assert.Equal(t, "2.0.0", result.LatestVersion)
	assert.True(t, result.HasUpdate)
	assert.Equal(t, "https://github.com/example/repo/releases/tag/v2.0.0", result.URL)
	assert.Greater(t, result.CheckedAt, int64(0))
}

func TestCheckForUpdates_AlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v1.0.0",
			"html_url": "https://github.com/example/repo/releases/tag/v1.0.0",
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := checkForUpdatesURL(ctx, "1.0.0", srv.URL+"/repos/example/repo/releases/latest")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1.0.0", result.LatestVersion)
	assert.False(t, result.HasUpdate)
}
