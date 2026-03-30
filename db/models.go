package db

import "time"

func init() {
	RegisterModel(&ProfileModel{})
	RegisterModel(&GroupModel{})
	RegisterModel(&SnippetModel{})
	RegisterModel(&AiRoleModel{})
}

// ProfileModel persists workspace profiles to the database.
type ProfileModel struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProfileKey string    `gorm:"size:128;uniqueIndex" json:"profile_key"`
	Name       string    `gorm:"size:256" json:"name"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GroupModel persists session groups to the database.
type GroupModel struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	GroupName  string    `gorm:"size:256" json:"group_name"`
	SortOrder  int       `json:"sort_order"`
	ProfileKey string    `gorm:"size:128;index" json:"profile_key"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SnippetModel persists command snippets to the database.
type SnippetModel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Index     int       `json:"index"`
	Name      string    `gorm:"size:256" json:"name"`
	Command   string    `gorm:"type:text" json:"command"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AiRoleModel persists custom AI roles to the database.
type AiRoleModel struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:128;uniqueIndex" json:"name"`
	Description  string    `gorm:"size:512" json:"description"`
	SystemPrompt string    `gorm:"type:text" json:"system_prompt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TaskSegment represents a single AI task within a terminal pane.
type TaskSegment struct {
	ID          string     `gorm:"primaryKey;size:32" json:"id"`
	Year        int        `gorm:"index:idx_seg_year_mon" json:"year"`
	Mon         int        `gorm:"index:idx_seg_year_mon" json:"mon"`
	Token       string     `gorm:"size:128;index:idx_seg_session" json:"token"`
	SessionName string     `gorm:"size:128;index:idx_seg_session" json:"session_name"`
	WindowIndex int        `gorm:"index:idx_seg_pane" json:"window_index"`
	WindowName  string     `gorm:"size:128" json:"window_name"`
	PaneIndex   int        `gorm:"index:idx_seg_pane" json:"pane_index"`
	TaskTitle   string     `gorm:"size:256" json:"task_title"`
	TaskStatus  string     `gorm:"size:32;default:in_progress" json:"task_status"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ChatMessage represents a conversation message within a task segment.
type ChatMessage struct {
	ID        string     `gorm:"primaryKey;size:32" json:"id"`
	Year      int        `gorm:"index:idx_msg_year_mon" json:"year"`
	Mon       int        `gorm:"index:idx_msg_year_mon" json:"mon"`
	SegmentID string     `gorm:"size:32;index:idx_msg_segment" json:"segment_id"`
	Role      string     `gorm:"size:32" json:"role"`
	Content   string     `gorm:"type:text" json:"content"`
	MsgTime   *time.Time `json:"msg_time"`
	CreatedAt time.Time  `json:"created_at"`
}

// CommandRecord represents a terminal command executed during a task segment.
type CommandRecord struct {
	ID        string     `gorm:"primaryKey;size:32" json:"id"`
	Year      int        `gorm:"index:idx_cmd_year_mon" json:"year"`
	Mon       int        `gorm:"index:idx_cmd_year_mon" json:"mon"`
	SegmentID string     `gorm:"size:32;index:idx_cmd_segment" json:"segment_id"`
	Command   string     `gorm:"type:text" json:"command"`
	CmdTime   *time.Time `json:"cmd_time"`
	ExitCode  int        `json:"exit_code"`
	CreatedAt time.Time  `json:"created_at"`
}

// TaskSummary represents a generated summary for a completed task segment.
type TaskSummary struct {
	ID             string     `gorm:"primaryKey;size:32" json:"id"`
	Year           int        `gorm:"index:idx_sum_year_mon" json:"year"`
	Mon            int        `gorm:"index:idx_sum_year_mon" json:"mon"`
	SegmentID      string     `gorm:"size:32;index:idx_sum_segment" json:"segment_id"`
	SessionName    string     `gorm:"size:128;index:idx_sum_session_window" json:"session_name"`
	WindowIndex    int        `gorm:"index:idx_sum_session_window" json:"window_index"`
	WindowName     string     `gorm:"size:128" json:"window_name"`
	CommandSummary string     `gorm:"type:text" json:"command_summary"`
	OutputSummary  string     `gorm:"type:text" json:"output_summary"`
	SummaryStatus  string     `gorm:"size:32;default:pending" json:"summary_status"`
	GeneratedAt    *time.Time `json:"generated_at"`
	CreatedAt      time.Time  `json:"created_at"`
}
