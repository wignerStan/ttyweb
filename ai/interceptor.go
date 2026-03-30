package ai

import (
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

// Metadata represents a side-channel message from the output interceptor.
type Metadata struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// OutputInterceptor observes terminal output without modifying it.
type OutputInterceptor interface {
	Intercept(data []byte) []Metadata
}

type detectionPattern struct {
	state   string
	pattern *regexp.Regexp
}

// ANSITerminalInterceptor strips ANSI escape sequences and matches
// AI assistant state patterns against the cleaned output.
type ANSITerminalInterceptor struct {
	mu           sync.RWMutex
	patterns     []detectionPattern
	workingPanes map[string]bool
}

// NewANSITerminalInterceptor creates an interceptor with default patterns.
func NewANSITerminalInterceptor() *ANSITerminalInterceptor {
	return &ANSITerminalInterceptor{
		workingPanes: make(map[string]bool),
		patterns: []detectionPattern{
			{state: "working", pattern: regexp.MustCompile(`(?m)^Use\s+\w+`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^Applying\s+changes`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^Reading\s+file`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^Editing\s+file`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^Writing\s+file`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^\[.*\]\s+(analyzing|processing|executing|running|building|compiling)`)},
			{state: "working", pattern: regexp.MustCompile(`(?m)^\s*[▉▊▋▌▍▎▏█]+`)},
			{state: "waiting_approval", pattern: regexp.MustCompile(`(?i)(need.?(s)?approval|waiting.?(for)?.?approval)`)},
			{state: "waiting_approval", pattern: regexp.MustCompile(`(?i)allow\s+\w+\s+to\s+execute`)},
			{state: "waiting_approval", pattern: regexp.MustCompile(`(?i)\[.*\]\s+approval\s+required`)},
			{state: "waiting_approval", pattern: regexp.MustCompile(`(?i)permit\s+action`)},
			{state: "idle", pattern: regexp.MustCompile(`(?m)^Human:\s*$`)},
		},
	}
}

// Intercept strips ANSI from data, matches patterns, returns metadata.
func (i *ANSITerminalInterceptor) Intercept(data []byte) []Metadata {
	if len(data) == 0 {
		return nil
	}
	stripped := ansi.Strip(string(data))
	var results []Metadata

	for _, dp := range i.patterns {
		if dp.pattern.MatchString(stripped) {
			results = append(results, Metadata{
				Type: "ai_state_change",
				Data: map[string]any{"state": dp.state},
			})
			break
		}
	}

	// Track working state for tab rename
	for _, m := range results {
		if m.Type == "ai_state_change" {
			i.mu.Lock()
			switch m.Data["state"] {
			case "working":
				i.workingPanes["*"] = true
			case "idle":
				delete(i.workingPanes, "*")
			}
			i.mu.Unlock()
		}
	}

	// If in working state and no pattern matched, check for user input
	i.mu.RLock()
	inWorking := i.workingPanes["*"]
	i.mu.RUnlock()

	if inWorking && len(results) == 0 {
		line := strings.TrimSpace(stripped)
		if line != "" && !looksLikeToolOutput(line) {
			results = append(results, Metadata{
				Type: "tab_rename",
				Data: map[string]any{"summary": truncate(line, 64)},
			})
		}
	}

	return results
}

// looksLikeToolOutput returns true if the line resembles tool/agent output
// rather than user input.
func looksLikeToolOutput(line string) bool {
	toolPrefixes := []string{
		"Use ", "Applying ", "Reading ", "Editing ", "Writing ",
		"[", "#", "$",
	}
	for _, prefix := range toolPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// truncate shortens s to at most maxLen runes, appending an ellipsis
// if truncation occurred.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "\u2026"
}
