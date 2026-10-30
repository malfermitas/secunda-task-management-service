package domain

import (
	"fmt"
	"strings"
	"time"

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
	ID          int64
	TeamID      int64
	Title       string
	Description *string
	Status      Status
	CreatedBy   int64
	AssigneeID  *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	Version     uint
}

// TaskUpdate contains fields to change on a task. Nil fields are unchanged.
type TaskUpdate struct {
	Title       *string
	Description *string
	Status      *Status
	AssigneeID  *int64
}

// NewTask creates a validated task entity in the todo state.
func NewTask(teamID int64, title string, description *string, createdBy int64, assigneeID *int64, now time.Time) (*Task, error) {
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
	if assigneeID != nil && *assigneeID <= 0 {
		return nil, errs.Validation("assignee id must be positive")
	}

	now = now.UTC()
	return &Task{
		TeamID:      teamID,
		Title:       title,
		Description: copyPtr(description),
		Status:      StatusTodo,
		CreatedBy:   createdBy,
		AssigneeID:  copyPtr(assigneeID),
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     0,
	}, nil
}

// ApplyUpdate validates and applies changes to the task, returning the
// field-level diff for history. It returns ErrNoChanges when nothing differs.
func (t *Task) ApplyUpdate(update TaskUpdate, now time.Time) (map[string]any, error) {
	changes := make(map[string]any)

	if update.Title != nil {
		title := strings.TrimSpace(*update.Title)
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

	if update.Description != nil && !equalPtr(t.Description, update.Description) {
		changes["description"] = fieldChange(deref(t.Description), deref(update.Description))
		t.Description = copyPtr(update.Description)
	}

	if update.Status != nil && *update.Status != t.Status {
		if !update.Status.Valid() {
			return nil, ErrInvalidStatus(string(*update.Status))
		}
		changes["status"] = fieldChange(string(t.Status), string(*update.Status))
		t.Status = *update.Status
		if t.Status == StatusDone {
			closedAt := now.UTC()
			t.ClosedAt = &closedAt
		} else {
			t.ClosedAt = nil
		}
	}

	if update.AssigneeID != nil && !equalPtr(t.AssigneeID, update.AssigneeID) {
		if *update.AssigneeID <= 0 {
			return nil, errs.Validation("assignee id must be positive")
		}
		changes["assignee_id"] = fieldChange(deref(t.AssigneeID), deref(update.AssigneeID))
		t.AssigneeID = copyPtr(update.AssigneeID)
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

func copyPtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}
