package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ttyweb/ai"
	"ttyweb/config"
)

// commitMessageSystemPrompt instructs the LLM to produce a conventional commit message.
const commitMessageSystemPrompt = `You are a commit message generator. Given a git diff, produce a single conventional commit message.

Rules:
- Format: <type>: <subject>
- Types: feat, fix, refactor, docs, test, chore, perf, ci
- Subject must be under 72 characters
- Use imperative mood (e.g. "add feature" not "added feature")
- No trailing period on subject
- If the diff is ambiguous, choose the most likely type
- Output ONLY the commit message, nothing else`

// CommitMessageService generates commit messages using an LLM.
type CommitMessageService struct {
	client *ai.Client
}

// NewCommitMessageService creates a CommitMessageService. If LLM is not
// configured (missing API key), the client is left nil and GenerateCommitMessage
// will return an error.
func NewCommitMessageService(cfg *config.Config) *CommitMessageService {
	if cfg == nil || strings.TrimSpace(cfg.LLM.APIKey) == "" {
		return &CommitMessageService{client: nil}
	}
	return &CommitMessageService{
		client: ai.NewClient(cfg.LLM.APIKey, cfg.LLM.APIURL, cfg.LLM.Model),
	}
}

// GenerateCommitMessage takes a git diff and returns an AI-generated conventional
// commit message.
func (s *CommitMessageService) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	diff = strings.TrimSpace(diff)
	if diff == "" {
		return "", errors.New("empty diff: no changes to generate a commit message for")
	}
	if s.client == nil {
		return "", errors.New("AI client not configured: set LLM_API_KEY in config or environment")
	}

	prompt := buildCommitPrompt(diff)
	message, err := s.client.ChatCompletion(ctx, commitMessageSystemPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("generate commit message: %w", err)
	}

	return strings.TrimSpace(message), nil
}

// buildCommitPrompt constructs the user prompt containing the diff for the LLM.
func buildCommitPrompt(diff string) string {
	return fmt.Sprintf("Generate a conventional commit message for the following diff:\n\n%s", diff)
}
