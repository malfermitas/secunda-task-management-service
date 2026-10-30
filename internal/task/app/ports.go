package app

import (
	"context"

	taskdomain "secunda-task-management-service/internal/task/domain"
)

// CreateInput contains data required to create a task.
type CreateInput struct {
	TeamID      int64
	Title       string
	Description *string
	AssigneeID  *int64
}

// UpdateInput contains fields to update on a task. Nil fields are unchanged.
type UpdateInput struct {
	Title       *string
	Description *string
	Status      *taskdomain.Status
	AssigneeID  *int64
	Version     *uint
}

// ListInput contains filters and pagination for listing team tasks.
type ListInput struct {
	TeamID     int64
	Status     *taskdomain.Status
	AssigneeID *int64
	Limit      int
	Offset     int
}

// TaskList is a read model returned by task listing.
type TaskList struct {
	Items  []taskdomain.Task
	Total  *int64
	Limit  int
	Offset int
}

// Cache is the application port for caching team task lists.
// Implementations must be fail-open and must not return errors.
type Cache interface {
	GetTasks(ctx context.Context, teamID int64) (*TaskList, bool)
	SetTasks(ctx context.Context, teamID int64, list *TaskList)
	InvalidateTeam(ctx context.Context, teamID int64)
}

// Service exposes task use cases.
type Service interface {
	Create(ctx context.Context, actorID int64, in CreateInput) (*taskdomain.Task, error)
	List(ctx context.Context, actorID int64, in ListInput) (*TaskList, error)
	Update(ctx context.Context, actorID, taskID int64, in UpdateInput) (*taskdomain.Task, error)
	History(ctx context.Context, actorID, taskID int64) ([]taskdomain.HistoryEntry, error)
	AddComment(ctx context.Context, actorID, taskID int64, content string) (*taskdomain.Comment, error)
	ListComments(ctx context.Context, actorID, taskID int64) ([]taskdomain.Comment, error)
}
