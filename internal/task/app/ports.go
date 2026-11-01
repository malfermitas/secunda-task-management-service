package app

import (
	"context"
	"fmt"
	"strconv"

	"secunda-task-management-service/internal/shared"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

// CreateInput contains data required to create a task.
type CreateInput struct {
	TeamID      int64
	Title       string
	Description shared.Optional[string]
	AssigneeID  shared.Optional[int64]
}

// UpdateInput contains fields to update on a task. None fields are unchanged.
type UpdateInput struct {
	Title       shared.Optional[string]
	Description shared.Optional[string]
	Status      shared.Optional[taskdomain.Status]
	AssigneeID  shared.Optional[int64]
	Version     shared.Optional[uint]
}

// ListInput contains filters and pagination for listing team tasks.
type ListInput struct {
	TeamID     int64
	Status     shared.Optional[taskdomain.Status]
	AssigneeID shared.Optional[int64]
	Limit      int
	Offset     int
}

// FilterKey returns the cache-key fragment for the list filters.
func (l ListInput) FilterKey() string {
	status := "all"
	if value, ok := l.Status.TryUnwrap(); ok {
		status = string(value)
	}

	assignee := "all"
	if value, ok := l.AssigneeID.TryUnwrap(); ok {
		assignee = strconv.FormatInt(value, 10)
	}

	return fmt.Sprintf(
		"status:%s:assignee:%s:limit:%d:offset:%d",
		status, assignee, l.Limit, l.Offset,
	)
}

// TaskList is a read model returned by task listing.
type TaskList struct {
	Items  []taskdomain.Task      `json:"items"`
	Total  shared.Optional[int64] `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// CacheKey is the opaque page key returned by Cache.GetTasks.
type CacheKey string

// Cache is the application port for caching team task lists.
// Implementations must be fail-open and must not return errors.
//
// GetTasks returns the page key observed for the lookup. Pass that key to
// SetTasks so a slow read cannot repopulate a generation that was invalidated
// while the database query was running.
type Cache interface {
	GetTasks(ctx context.Context, teamID int64, filterKey string) (*TaskList, CacheKey, bool)
	SetTasks(ctx context.Context, teamID int64, key CacheKey, list *TaskList)
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
