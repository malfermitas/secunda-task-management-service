package domain

import (
	"context"

	"secunda-task-management-service/internal/shared"
)

// TaskStatusCount is an aggregate of tasks per status.
type TaskStatusCount struct {
	Status string `json:"status" enums:"todo,in_progress,done" example:"todo"`
	Count  int64  `json:"count" example:"5"`
}

// TopAssignee is an aggregate of closed tasks per assignee.
type TopAssignee struct {
	UserID      int64  `json:"user_id" example:"5"`
	Name        string `json:"name" example:"Jane Doe"`
	ClosedCount int64  `json:"closed_count" example:"4"`
}

// TaskCommentCount is the number of comments per task.
type TaskCommentCount struct {
	TaskID int64 `json:"task_id" example:"10"`
	Count  int64 `json:"count" example:"3"`
}

// TeamStats is the analytical report for a single team.
type TeamStats struct {
	TeamID          int64                    `json:"team_id" example:"1"`
	ByStatus        []TaskStatusCount        `json:"by_status"`
	TopAssignees    []TopAssignee            `json:"top_assignees"`
	AvgCloseSeconds shared.Optional[float64] `json:"avg_close_seconds" swaggertype:"number" example:"3600"`
	CommentsPerTask []TaskCommentCount       `json:"comments_per_task"`
}

// Repository is the read port for team analytics.
type Repository interface {
	TeamStats(ctx context.Context, teamID int64) (*TeamStats, error)
}
