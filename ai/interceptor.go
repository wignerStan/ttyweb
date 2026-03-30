package ai

import (
	"regexp"
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
	mu       sync.RWMutex
	patterns []detectionPattern
}

// NewANSITerminalInterceptor creates an interceptor with default patterns.
func NewANSITerminalInterceptor() *ANSITerminalInterceptor {
	return &ANSITerminalInterceptor{
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

	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, dp := range i.patterns {
		if dp.pattern.MatchString(stripped) {
			results = append(results, Metadata{
				Type: "ai_state_change",
				Data: map[string]any{"state": dp.state},
			})
			break
		}
	}
	return results
}
