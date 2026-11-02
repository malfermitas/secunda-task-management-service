package dto

import (
	"time"

	analyticsdomain "secunda-task-management-service/internal/analytics/domain"
	"secunda-task-management-service/internal/shared"
	taskdomain "secunda-task-management-service/internal/task/domain"
	teamdomain "secunda-task-management-service/internal/team/domain"
	userdomain "secunda-task-management-service/internal/user/domain"
)

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        int64     `json:"id" example:"1"`
	Email     string    `json:"email" example:"jane@example.com"`
	Name      string    `json:"name" example:"Jane Doe"`
	CreatedAt time.Time `json:"created_at" example:"2026-11-01T10:00:00Z"`
}

// NewUserResponse maps a user entity to a response.
func NewUserResponse(user *userdomain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email.String(),
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	}
}

// TeamResponse is the public representation of a team.
type TeamResponse struct {
	ID        int64     `json:"id" example:"1"`
	Name      string    `json:"name" example:"Platform Team"`
	CreatedBy int64     `json:"created_by" example:"1"`
	CreatedAt time.Time `json:"created_at" example:"2026-11-01T10:00:00Z"`
}

// NewTeamResponse maps a team entity to a response.
func NewTeamResponse(team teamdomain.Team) TeamResponse {
	return TeamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
		CreatedAt: team.CreatedAt,
	}
}

// NewTeamResponses maps a team slice to responses.
func NewTeamResponses(teams []teamdomain.Team) []TeamResponse {
	responses := make([]TeamResponse, 0, len(teams))
	for _, team := range teams {
		responses = append(responses, NewTeamResponse(team))
	}
	return responses
}

// TaskResponse is the public representation of a task.
type TaskResponse struct {
	ID          int64                      `json:"id" example:"10"`
	TeamID      int64                      `json:"team_id" example:"1"`
	Title       string                     `json:"title" example:"Fix login"`
	Description shared.Optional[string]    `json:"description" swaggertype:"string" example:"Users cannot log in with SSO" extensions:"x-nullable"`
	Status      string                     `json:"status" enums:"todo,in_progress,done" example:"todo"`
	CreatedBy   int64                      `json:"created_by" example:"1"`
	AssigneeID  shared.Optional[int64]     `json:"assignee_id" swaggertype:"integer" example:"5" extensions:"x-nullable"`
	CreatedAt   time.Time                  `json:"created_at" example:"2026-11-01T10:00:00Z"`
	UpdatedAt   time.Time                  `json:"updated_at" example:"2026-11-01T10:00:00Z"`
	ClosedAt    shared.Optional[time.Time] `json:"closed_at" swaggertype:"string" format:"date-time" example:"2026-11-01T12:00:00Z" extensions:"x-nullable"`
	Version     uint                       `json:"version" example:"1"`
}

// NewTaskResponse maps a task entity to a response.
func NewTaskResponse(task *taskdomain.Task) TaskResponse {
	return TaskResponse{
		ID:          task.ID,
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		CreatedBy:   task.CreatedBy,
		AssigneeID:  task.AssigneeID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		ClosedAt:    task.ClosedAt,
		Version:     task.Version,
	}
}

// NewTaskResponses maps a task slice to responses.
func NewTaskResponses(tasks []taskdomain.Task) []TaskResponse {
	responses := make([]TaskResponse, 0, len(tasks))
	for i := range tasks {
		responses = append(responses, NewTaskResponse(&tasks[i]))
	}
	return responses
}

// HistoryEntryResponse is the public representation of a task history entry.
type HistoryEntryResponse struct {
	ID        int64          `json:"id" example:"1"`
	TaskID    int64          `json:"task_id" example:"10"`
	ChangedBy int64          `json:"changed_by" example:"1"`
	Changes   map[string]any `json:"changes" swaggertype:"object"`
	CreatedAt time.Time      `json:"created_at" example:"2026-11-01T10:00:00Z"`
}

// NewHistoryEntryResponse maps a history entry to a response.
func NewHistoryEntryResponse(entry taskdomain.HistoryEntry) HistoryEntryResponse {
	return HistoryEntryResponse{
		ID:        entry.ID,
		TaskID:    entry.TaskID,
		ChangedBy: entry.ChangedBy,
		Changes:   entry.Changes,
		CreatedAt: entry.CreatedAt,
	}
}

// NewHistoryEntryResponses maps history entries to responses.
func NewHistoryEntryResponses(entries []taskdomain.HistoryEntry) []HistoryEntryResponse {
	responses := make([]HistoryEntryResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, NewHistoryEntryResponse(entry))
	}
	return responses
}

// TaskStatusCountResponse is the number of tasks in one status.
type TaskStatusCountResponse struct {
	Status string `json:"status" enums:"todo,in_progress,done" example:"todo"`
	Count  int64  `json:"count" example:"5"`
}

// TopAssigneeResponse is a top assignee aggregate.
type TopAssigneeResponse struct {
	UserID      int64  `json:"user_id" example:"5"`
	Name        string `json:"name" example:"Jane Doe"`
	ClosedCount int64  `json:"closed_count" example:"4"`
}

// TaskCommentCountResponse is the number of comments on a task.
type TaskCommentCountResponse struct {
	TaskID int64 `json:"task_id" example:"10"`
	Count  int64 `json:"count" example:"3"`
}

// TeamStatsResponse is the public analytics report.
type TeamStatsResponse struct {
	TeamID          int64                      `json:"team_id" example:"1"`
	ByStatus        []TaskStatusCountResponse  `json:"by_status"`
	TopAssignees    []TopAssigneeResponse      `json:"top_assignees"`
	AvgCloseSeconds shared.Optional[float64]   `json:"avg_close_seconds" swaggertype:"number" example:"3600"`
	CommentsPerTask []TaskCommentCountResponse `json:"comments_per_task"`
}

// NewTeamStatsResponse maps the analytics domain report to a response.
func NewTeamStatsResponse(stats *analyticsdomain.TeamStats) TeamStatsResponse {
	byStatus := make([]TaskStatusCountResponse, 0, len(stats.ByStatus))
	for _, item := range stats.ByStatus {
		byStatus = append(byStatus, TaskStatusCountResponse{
			Status: item.Status,
			Count:  item.Count,
		})
	}

	topAssignees := make([]TopAssigneeResponse, 0, len(stats.TopAssignees))
	for _, item := range stats.TopAssignees {
		topAssignees = append(topAssignees, TopAssigneeResponse{
			UserID:      item.UserID,
			Name:        item.Name,
			ClosedCount: item.ClosedCount,
		})
	}

	commentsPerTask := make([]TaskCommentCountResponse, 0, len(stats.CommentsPerTask))
	for _, item := range stats.CommentsPerTask {
		commentsPerTask = append(commentsPerTask, TaskCommentCountResponse{
			TaskID: item.TaskID,
			Count:  item.Count,
		})
	}

	return TeamStatsResponse{
		TeamID:          stats.TeamID,
		ByStatus:        byStatus,
		TopAssignees:    topAssignees,
		AvgCloseSeconds: stats.AvgCloseSeconds,
		CommentsPerTask: commentsPerTask,
	}
}

// CommentResponse is the public representation of a task comment.
type CommentResponse struct {
	ID        int64     `json:"id" example:"1"`
	TaskID    int64     `json:"task_id" example:"10"`
	UserID    int64     `json:"user_id" example:"2"`
	Content   string    `json:"content" example:"I reproduced this on staging."`
	CreatedAt time.Time `json:"created_at" example:"2026-11-01T10:00:00Z"`
}

// NewCommentResponse maps a comment to a response.
func NewCommentResponse(comment *taskdomain.Comment) CommentResponse {
	return CommentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}
}

// NewCommentResponses maps comments to responses.
func NewCommentResponses(comments []taskdomain.Comment) []CommentResponse {
	responses := make([]CommentResponse, 0, len(comments))
	for i := range comments {
		responses = append(responses, NewCommentResponse(&comments[i]))
	}
	return responses
}
