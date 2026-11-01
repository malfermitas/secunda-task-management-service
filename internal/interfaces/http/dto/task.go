package dto

import taskdomain "secunda-task-management-service/internal/task/domain"

// CreateTaskRequest is the payload for creating a task.
type CreateTaskRequest struct {
	TeamID      int64   `json:"team_id" binding:"required,gt=0"`
	Title       string  `json:"title" binding:"required,min=1,max=255"`
	Description *string `json:"description"`
	AssigneeID  *int64  `json:"assignee_id" binding:"omitempty,gt=0"`
}

// UpdateTaskRequest is the payload for updating a task.
// Nil fields are left unchanged.
type UpdateTaskRequest struct {
	Title       *string            `json:"title" binding:"omitempty,min=1,max=255"`
	Description *string            `json:"description"`
	Status      *taskdomain.Status `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID  *int64             `json:"assignee_id" binding:"omitempty,gt=0"`
	Version     *uint              `json:"version"`
}

// ListTasksQuery contains filters and pagination for GET /tasks.
type ListTasksQuery struct {
	TeamID     int64             `form:"team_id" binding:"required,gt=0"`
	Status     taskdomain.Status `form:"status" binding:"omitempty,oneof=todo in_progress done"`
	AssigneeID *int64            `form:"assignee_id" binding:"omitempty,gt=0"`
	Limit      int               `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset     int               `form:"offset" binding:"omitempty,min=0"`
}

// TaskListResponse is the response for GET /tasks.
type TaskListResponse struct {
	Items  []TaskResponse `json:"items"`
	Total  *int64         `json:"total,omitempty"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// CreateCommentRequest is the payload for adding a task comment.
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000"`
}
