package worktree


// Operation error kinds.
const (
	KindWorktreeLocked = "worktree_locked"
	KindConflict       = "conflict"
	KindNotFound       = "not_found"
	KindGitFailed      = "git_failed"
	KindAlreadyExists  = "already_exists"
	KindValidation     = "validation"
	KindCancelled      = "cancelled"
)

// Sentinel errors for use with errors.Is.
var (
	ErrWorktreeLocked = &kindError{kind: KindWorktreeLocked}
	ErrConflict       = &kindError{kind: KindConflict}
	ErrNotFound       = &kindError{kind: KindNotFound}
	ErrGitFailed      = &kindError{kind: KindGitFailed}
	ErrAlreadyExists  = &kindError{kind: KindAlreadyExists}
	ErrValidation     = &kindError{kind: KindValidation}
	ErrCancelled      = &kindError{kind: KindCancelled}
)

// kindError is a sentinel error that identifies an error kind.
type kindError struct {
	kind string
}

func (e *kindError) Error() string { return e.kind }

// OpError is a structured error for git worktree operations.
type OpError struct {
	Kind   string // One of the Kind* constants above.
	Path   string // Repository or worktree path involved.
	Detail string // Human-readable explanation.
	Cause  error  // Underlying error, if any.
}

func (e *OpError) Error() string {
	msg := e.Kind + ": " + e.Path
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	return msg
}

// Is matches this OpError against a sentinel kindError target.
func (e *OpError) Is(target error) bool {
	t, ok := target.(*kindError)
	return ok && t.kind == e.Kind
}

// Unwrap returns the underlying cause for errors.Is/As chaining.
func (e *OpError) Unwrap() error {
	return e.Cause
}

