package dto

import (
	"time"

	taskdomain "secunda-task-management-service/internal/task/domain"
	teamdomain "secunda-task-management-service/internal/team/domain"
	userdomain "secunda-task-management-service/internal/user/domain"
)

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
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
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
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
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	AssigneeID  *int64     `json:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	Version     uint       `json:"version"`
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
	ID        int64          `json:"id"`
	TaskID    int64          `json:"task_id"`
	ChangedBy int64          `json:"changed_by"`
	Changes   map[string]any `json:"changes"`
	CreatedAt time.Time      `json:"created_at"`
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

// CommentResponse is the public representation of a task comment.
type CommentResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
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
