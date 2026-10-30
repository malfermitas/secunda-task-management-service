package domain

import "time"

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// TaskComment represents a comment on a task.
type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskHistoryEntry represents a change in a task's history.
type TaskHistoryEntry struct {
	ID        int64          `json:"id"`
	TaskID    int64          `json:"task_id"`
	ChangedBy int64          `json:"changed_by"`
	Changes   map[string]any `json:"changes"`
	CreatedAt time.Time      `json:"created_at"`
}

// Task represents a task within a team.
type Task struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	AssigneeID  *int64     `json:"assignee_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	Version     uint       `json:"version"`
}
