package db

// TaskAISession represents a many-to-many link between a kanban task
// and an AI session.
type TaskAISession struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	AISessionID string `json:"ai_session_id"`
}

func init() {
	RegisterModel(&TaskAISession{})
}
