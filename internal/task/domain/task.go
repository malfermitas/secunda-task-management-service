package domain

import (
	"fmt"
	"strings"
	"time"

	"secunda-task-management-service/internal/shared"
	"secunda-task-management-service/internal/shared/errs"
)

// Status is a value object describing the task lifecycle state.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Valid reports whether the status is supported.
func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

// Task is the task aggregate root.
type Task struct {
	ID          int64                      `json:"id"`
	TeamID      int64                      `json:"team_id"`
	Title       string                     `json:"title"`
	Description shared.Optional[string]    `json:"description"`
	Status      Status                     `json:"status"`
	CreatedBy   int64                      `json:"created_by"`
	AssigneeID  shared.Optional[int64]     `json:"assignee_id"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	ClosedAt    shared.Optional[time.Time] `json:"closed_at"`
	Version     uint                       `json:"version"`
}

// TaskUpdate contains fields to change on a task. None fields are unchanged.
type TaskUpdate struct {
	Title       shared.Optional[string]
	Description shared.Optional[string]
	Status      shared.Optional[Status]
	AssigneeID  shared.Optional[int64]
}

// NewTask creates a validated task entity in the todo state.
func NewTask(teamID int64, title string, description shared.Optional[string], createdBy int64, assigneeID shared.Optional[int64], now time.Time) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errs.Validation("task title is required")
	}
	if len(title) > 255 {
		return nil, errs.Validation("task title must be at most 255 characters")
	}
	if teamID <= 0 {
		return nil, errs.Validation("team id is required")
	}
	if createdBy <= 0 {
		return nil, errs.Validation("creator id is required")
	}
	if value, ok := assigneeID.TryUnwrap(); ok && value <= 0 {
		return nil, errs.Validation("assignee id must be positive")
	}

	now = now.UTC()
	return &Task{
		TeamID:      teamID,
		Title:       title,
		Description: description,
		Status:      StatusTodo,
		CreatedBy:   createdBy,
		AssigneeID:  assigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     0,
	}, nil
}

// ApplyUpdate validates and applies changes to the task, returning the
// field-level diff for history. It returns ErrNoChanges when nothing differs.
func (t *Task) ApplyUpdate(update TaskUpdate, now time.Time) (map[string]any, error) {
	changes := make(map[string]any)

	if value, ok := update.Title.TryUnwrap(); ok {
		title := strings.TrimSpace(value)
		if title == "" {
			return nil, errs.Validation("task title is required")
		}
		if len(title) > 255 {
			return nil, errs.Validation("task title must be at most 255 characters")
		}
		if title != t.Title {
			changes["title"] = fieldChange(t.Title, title)
			t.Title = title
		}
	}

	if value, ok := update.Description.TryUnwrap(); ok {
		if old, oldOK := t.Description.TryUnwrap(); !oldOK || old != value {
			changes["description"] = fieldChange(old, value)
			t.Description = shared.Some(value)
		}
	}

	if value, ok := update.Status.TryUnwrap(); ok && value != t.Status {
		if !value.Valid() {
			return nil, ErrInvalidStatus(string(value))
		}
		changes["status"] = fieldChange(string(t.Status), string(value))
		t.Status = value

		oldClosedAt := t.ClosedAt
		if t.Status == StatusDone {
			t.ClosedAt = shared.Some(now.UTC())
		} else {
			t.ClosedAt = shared.None[time.Time]()
		}
		if !oldClosedAt.Equal(t.ClosedAt) {
			changes["closed_at"] = fieldChange(oldClosedAt, t.ClosedAt)
		}
	}

	if value, ok := update.AssigneeID.TryUnwrap(); ok {
		if value <= 0 {
			return nil, errs.Validation("assignee id must be positive")
		}
		if old, oldOK := t.AssigneeID.TryUnwrap(); !oldOK || old != value {
			changes["assignee_id"] = fieldChange(t.AssigneeID, value)
			t.AssigneeID = shared.Some(value)
		}
	}

	if len(changes) == 0 {
		return nil, ErrNoChanges()
	}

	now = now.UTC()
	t.UpdatedAt = now
	t.Version++
	return changes, nil
}

// HistoryEntry records a single task change set.
type HistoryEntry struct {
	ID        int64
	TaskID    int64
	ChangedBy int64
	Changes   map[string]any
	CreatedAt time.Time
}

// Comment is a task comment entity.
type Comment struct {
	ID        int64
	TaskID    int64
	UserID    int64
	Content   string
	CreatedAt time.Time
}

// NewComment creates a validated task comment.
func NewComment(taskID, userID int64, content string, now time.Time) (*Comment, error) {
	content = strings.TrimSpace(content)
	if taskID <= 0 || userID <= 0 {
		return nil, errs.Validation("task id and user id are required")
	}
	if content == "" {
		return nil, errs.Validation("comment content is required")
	}
	if len(content) > 10000 {
		return nil, errs.Validation("comment content is too long")
	}

	return &Comment{
		TaskID:    taskID,
		UserID:    userID,
		Content:   content,
		CreatedAt: now.UTC(),
	}, nil
}

// ErrTaskNotFound is returned when a task does not exist.
func ErrTaskNotFound(id int64) *errs.Error {
	return errs.NotFound("task_not_found", fmt.Sprintf("task %d not found", id))
}

// ErrInvalidStatus is returned when a status is not supported.
func ErrInvalidStatus(status string) *errs.Error {
	return errs.Validation(fmt.Sprintf("invalid status %q", status))
}

// ErrAssigneeNotMember is returned when assigning a non-member.
func ErrAssigneeNotMember(userID, teamID int64) *errs.Error {
	return errs.Validation(fmt.Sprintf("user %d is not a member of team %d", userID, teamID))
}

// ErrVersionConflict is returned when a task was updated concurrently.
func ErrVersionConflict() *errs.Error {
	return errs.Conflict("version_conflict", "task was modified concurrently")
}

// ErrNoChanges is returned when an update carries no effective changes.
func ErrNoChanges() *errs.Error {
	return errs.Validation("no fields to update")
}

func fieldChange(oldValue, newValue any) map[string]any {
	return map[string]any{"old": oldValue, "new": newValue}
}
