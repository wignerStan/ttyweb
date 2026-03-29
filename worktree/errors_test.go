package worktree

import (
	"errors"
	"testing"
)

func TestOpError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *OpError
		want string
	}{
		{
			name: "worktree locked",
			err:  &OpError{Kind: KindWorktreeLocked, Path: "/repo/wt", Detail: "another process"},
			want: "worktree_locked: /repo/wt (another process)",
		},
		{
			name: "conflict",
			err:  &OpError{Kind: KindConflict, Path: "/repo/wt", Detail: "merge conflict in main.go"},
			want: "conflict: /repo/wt (merge conflict in main.go)",
		},
		{
			name: "not found",
			err:  &OpError{Kind: KindNotFound, Path: "/repo/wt"},
			want: "not_found: /repo/wt",
		},
		{
			name: "empty kind",
			err:  &OpError{Kind: "unknown", Path: "/repo"},
			want: "unknown: /repo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.err.Error()
			if got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOpError_Is(t *testing.T) {
	t.Parallel()

	base := &OpError{Kind: KindWorktreeLocked, Path: "/repo/wt"}

	if !errors.Is(base, ErrWorktreeLocked) {
		t.Error("expected errors.Is to match ErrWorktreeLocked")
	}
	if errors.Is(base, ErrConflict) {
		t.Error("expected errors.Is NOT to match ErrConflict")
	}
	if errors.Is(errors.New("some error"), ErrWorktreeLocked) {
		t.Error("expected plain error NOT to match ErrWorktreeLocked")
	}
}

func TestOpError_Unwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("git exit status 128")
	err := &OpError{Kind: KindGitFailed, Path: "/repo", Detail: "branch create failed", Cause: inner}

	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to match inner cause via Unwrap")
	}
	var opErr *OpError
	if !errors.As(err, &opErr) {
		t.Error("expected errors.As to extract OpError")
	}
	if opErr.Kind != KindGitFailed {
		t.Errorf("expected Kind %q, got %q", KindGitFailed, opErr.Kind)
	}
}
