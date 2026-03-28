package ai

import "testing"

func TestBuiltinRolesCount(t *testing.T) {
	roles := BuiltinRoles()
	if len(roles) != 7 {
		t.Fatalf("expected 7 builtin roles, got %d", len(roles))
	}
}

func TestBuiltinRolesHaveRequiredFields(t *testing.T) {
	roles := BuiltinRoles()
	for _, r := range roles {
		if r.ID == "" {
			t.Errorf("role at index missing ID")
		}
		if r.Name == "" {
			t.Errorf("role %q missing Name", r.ID)
		}
		if r.SystemPrompt == "" {
			t.Errorf("role %q missing SystemPrompt", r.ID)
		}
	}
}

func TestBuiltinRoleIDs(t *testing.T) {
	roles := BuiltinRoles()
	expected := map[string]bool{
		"cli-expert":       false,
		"ops-expert":       false,
		"prompt-optimizer": false,
		"frontend-expert":  false,
		"backend-expert":   false,
		"ui-expert":        false,
		"api-converter":    false,
	}

	for _, r := range roles {
		if _, ok := expected[r.ID]; !ok {
			t.Errorf("unexpected role ID: %q", r.ID)
		}
		expected[r.ID] = true
	}

	for id, found := range expected {
		if !found {
			t.Errorf("missing expected role ID: %q", id)
		}
	}
}

func TestGetRole(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantNil bool
	}{
		{"cli-expert exists", "cli-expert", false},
		{"ops-expert exists", "ops-expert", false},
		{"prompt-optimizer exists", "prompt-optimizer", false},
		{"frontend-expert exists", "frontend-expert", false},
		{"backend-expert exists", "backend-expert", false},
		{"ui-expert exists", "ui-expert", false},
		{"api-converter exists", "api-converter", false},
		{"nonexistent returns nil", "nonexistent", true},
		{"empty string returns nil", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetRole(tt.id)
			if tt.wantNil && r != nil {
				t.Fatalf("expected nil for %q, got %+v", tt.id, r)
			}
			if !tt.wantNil && r == nil {
				t.Fatalf("expected role for %q, got nil", tt.id)
			}
			if !tt.wantNil {
				if r.ID != tt.id {
					t.Errorf("expected ID %q, got %q", tt.id, r.ID)
				}
			}
		})
	}
}

func TestGetRoleDefault(t *testing.T) {
	r := GetRoleDefault("nonexistent")
	if r == nil {
		t.Fatal("GetRoleDefault should never return nil")
	}
	if r.ID != "cli-expert" {
		t.Errorf("expected default role ID cli-expert, got %q", r.ID)
	}

	// Existing role should be returned directly.
	r2 := GetRoleDefault("ops-expert")
	if r2 == nil {
		t.Fatal("GetRoleDefault returned nil for existing role")
	}
	if r2.ID != "ops-expert" {
		t.Errorf("expected ops-expert, got %q", r2.ID)
	}
}

func TestGetRoleReturnsValidData(t *testing.T) {
	// Calling GetRole twice should return equal data, regardless of pointer identity.
	r1 := GetRole("cli-expert")
	r2 := GetRole("cli-expert")
	if r1.ID != r2.ID || r1.Name != r2.Name || r1.SystemPrompt != r2.SystemPrompt {
		t.Error("two lookups returned different role data")
	}
}
