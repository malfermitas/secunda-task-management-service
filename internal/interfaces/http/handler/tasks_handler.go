package handler

import (
	"net/http"

	"secunda-task-management-service/internal/interfaces/http/dto"
	"secunda-task-management-service/internal/shared"
	taskapp "secunda-task-management-service/internal/task/app"

	"github.com/gin-gonic/gin"
)

// TasksHandler handles task, history and comment endpoints.
type TasksHandler struct {
	svc taskapp.Service
}

// Create handles POST /api/v1/tasks.
func (h *TasksHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	task, err := h.svc.Create(c.Request.Context(), userID, taskapp.CreateInput{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewTaskResponse(task))
}

// List handles GET /api/v1/tasks.
func (h *TasksHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var query dto.ListTasksQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBindingError(c, err)
		return
	}

	in := taskapp.ListInput{
		TeamID:     query.TeamID,
		Status:     shared.SomeIf(query.Status, query.Status != ""),
		AssigneeID: query.AssigneeID,
		Limit:      query.Limit,
		Offset:     query.Offset,
	}

	result, err := h.svc.List(c.Request.Context(), userID, in)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TaskListResponse{
		Items:  dto.NewTaskResponses(result.Items),
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	})
}

// Update handles PUT /api/v1/tasks/:id.
func (h *TasksHandler) Update(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	taskID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	task, err := h.svc.Update(c.Request.Context(), userID, taskID, taskapp.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		Version:     req.Version,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewTaskResponse(task))
}

// History handles GET /api/v1/tasks/:id/history.
func (h *TasksHandler) History(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	taskID, ok := pathID(c, "id")
	if !ok {
		return
	}

	entries, err := h.svc.History(c.Request.Context(), userID, taskID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewHistoryEntryResponses(entries))
}

// AddComment handles POST /api/v1/tasks/:id/comments.
func (h *TasksHandler) AddComment(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	taskID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	comment, err := h.svc.AddComment(c.Request.Context(), userID, taskID, req.Content)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewCommentResponse(comment))
}

// ListComments handles GET /api/v1/tasks/:id/comments.
func (h *TasksHandler) ListComments(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	taskID, ok := pathID(c, "id")
	if !ok {
		return
	}

	comments, err := h.svc.ListComments(c.Request.Context(), userID, taskID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewCommentResponses(comments))
}
