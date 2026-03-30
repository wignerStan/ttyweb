package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UpdateCheck represents the result of a version check against GitHub.
type UpdateCheck struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	CheckedAt      int64  `json:"checked_at"`
	URL            string `json:"url,omitempty"`
}

// githubRelease represents a GitHub release API response.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdates compares the current version against the latest GitHub release.
func CheckForUpdates(ctx context.Context, currentVersion string) (*UpdateCheck, error) {
	const owner = "wignerStanley"
	const repo = "ttyweb"
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	return checkForUpdatesURL(ctx, currentVersion, releaseURL)
}

// checkForUpdatesURL performs the update check against the given URL.
// Extracted for testability.
func checkForUpdatesURL(ctx context.Context, currentVersion, releaseURL string) (*UpdateCheck, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No releases found - not an error.
		return &UpdateCheck{
			CurrentVersion: currentVersion,
			LatestVersion:  "",
			HasUpdate:      false,
			CheckedAt:      time.Now().Unix(),
		}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	hasUpdate := latestVersion != "" && latestVersion != currentVersion

	return &UpdateCheck{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		CheckedAt:      time.Now().Unix(),
		URL:            release.HTMLURL,
	}, nil
}
