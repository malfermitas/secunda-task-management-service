package domain

import "context"

// Filter describes task list criteria.
type Filter struct {
	TeamID     int64
	Status     *Status
	AssigneeID *int64
	Limit      int
	Offset     int
}

// Page is a page of tasks.
type Page struct {
	Items []Task
	Total int64
}

// Repository is the persistence port for the task aggregate.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, taskID int64) (*Task, error)
	List(ctx context.Context, filter Filter) (*Page, error)
	// Save persists the task and its history diff atomically, guarding against
	// concurrent updates with expectedVersion.
	Save(ctx context.Context, task *Task, expectedVersion uint, changes map[string]any) error
	History(ctx context.Context, taskID int64) ([]HistoryEntry, error)
	AddComment(ctx context.Context, comment *Comment) error
	Comments(ctx context.Context, taskID int64) ([]Comment, error)
}
