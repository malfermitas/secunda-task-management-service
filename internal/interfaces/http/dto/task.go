package dto

import (
	"secunda-task-management-service/internal/shared"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

// CreateTaskRequest is the payload for creating a task.
type CreateTaskRequest struct {
	TeamID      int64   `json:"team_id" binding:"required,gt=0" example:"1"`
	Title       string  `json:"title" binding:"required,min=1,max=255" example:"Fix login"`
	Description *string `json:"description" example:"Users cannot log in with SSO"`
	AssigneeID  *int64  `json:"assignee_id" binding:"omitempty,gt=0" example:"5"`
}

// UpdateTaskRequest is the payload for updating a task.
// Nil fields are left unchanged.
type UpdateTaskRequest struct {
	Title       *string            `json:"title" binding:"omitempty,min=1,max=255" example:"Fix login"`
	Description *string            `json:"description" example:"Updated description"`
	Status      *taskdomain.Status `json:"status" binding:"omitempty,oneof=todo in_progress done" enums:"todo,in_progress,done" example:"in_progress"`
	AssigneeID  *int64             `json:"assignee_id" binding:"omitempty,gt=0" example:"5"`
	Version     *uint              `json:"version" example:"1"`
}

// ListTasksQuery contains filters and pagination for GET /tasks.
type ListTasksQuery struct {
	TeamID     int64             `form:"team_id" binding:"required,gt=0" example:"1"`
	Status     taskdomain.Status `form:"status" binding:"omitempty,oneof=todo in_progress done" enums:"todo,in_progress,done" example:"todo"`
	AssigneeID *int64            `form:"assignee_id" binding:"omitempty,gt=0" example:"5"`
	Limit      int               `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Offset     int               `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// TaskListResponse is the response for GET /tasks.
type TaskListResponse struct {
	Items  []TaskResponse         `json:"items"`
	Total  shared.Optional[int64] `json:"total" swaggertype:"integer" example:"1" extensions:"x-nullable"`
	Limit  int                    `json:"limit" example:"20"`
	Offset int                    `json:"offset" example:"0"`
}

// CreateCommentRequest is the payload for adding a task comment.
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000" example:"I reproduced this on staging."`
}
