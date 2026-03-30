package db

import (
	"testing"
	"time"
)

func TestProfileModelFields(t *testing.T) {
	p := ProfileModel{
		ProfileKey: "test-profile",
		Name:       "Test Profile",
		SortOrder:  1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if p.ProfileKey != "test-profile" || p.Name != "Test Profile" {
		t.Error("fields not set correctly")
	}
}

func TestGroupModelFields(t *testing.T) {
	g := GroupModel{
		GroupName:  "test-group",
		SortOrder:  2,
		ProfileKey: "test-profile",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if g.ProfileKey != "test-profile" {
		t.Error("ProfileKey not set")
	}
}

func TestSnippetModelFields(t *testing.T) {
	s := SnippetModel{
		Index:   0,
		Name:    "list files",
		Command: "ls -la",
	}
	if s.Command != "ls -la" {
		t.Error("Command not set")
	}
}

func TestAiRoleModelFields(t *testing.T) {
	r := AiRoleModel{
		Name:         "Test Role",
		Description:  "A test role",
		SystemPrompt: "You are a test assistant.",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if r.SystemPrompt != "You are a test assistant." {
		t.Error("SystemPrompt not set")
	}
}
