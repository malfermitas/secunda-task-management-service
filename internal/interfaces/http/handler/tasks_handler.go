package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"secunda-task-management-service/internal/interfaces/http/dto"
	"secunda-task-management-service/internal/shared"
	sharedlogging "secunda-task-management-service/internal/shared/logging"
	taskapp "secunda-task-management-service/internal/task/app"
)

// TasksHandler handles task, history and comment endpoints.
type TasksHandler struct {
	svc taskapp.Service
}

// Create handles POST /api/v1/tasks.
//
//	@Summary		Create a task
//	@Description	Any team member can create a task in their team. The current user becomes the task creator. The assignee must be a member of the same team.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateTaskRequest	true	"Task payload"
//	@Success		201		{object}	dto.TaskResponse
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	dto.ErrorResponse	"User is not a team member"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error or assignee is not a team member"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks [post]
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
		Description: shared.FromPtr(req.Description),
		AssigneeID:  shared.FromPtr(req.AssigneeID),
	})
	if err != nil {
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"task created",
		zap.Int64("task_id", task.ID),
		zap.Int64("team_id", task.TeamID),
	)
	c.JSON(http.StatusCreated, dto.NewTaskResponse(task))
}

// List handles GET /api/v1/tasks.
//
//	@Summary		List team tasks
//	@Description	Requires team_id and team membership. The list is cached in Redis for 5 minutes per team and filter set.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Produce		json
//	@Param			team_id		query		int		true	"Team ID"			example(1)
//	@Param			status		query		string	false	"Task status"		Enums(todo,in_progress,done)
//	@Param			assignee_id	query		int		false	"Assignee user ID"	example(5)
//	@Param			limit		query		int		false	"Page size"			default(20)	minimum(1)	maximum(100)
//	@Param			offset		query		int		false	"Page offset"		default(0)	minimum(0)
//	@Success		200			{object}	dto.TaskListResponse
//	@Failure		400			{object}	dto.ErrorResponse	"Invalid query parameters"
//	@Failure		401			{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403			{object}	dto.ErrorResponse	"User is not a team member"
//	@Failure		422			{object}	dto.ErrorResponse	"Invalid status"
//	@Failure		500			{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks [get]
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
		AssigneeID: shared.FromPtr(query.AssigneeID),
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
//
//	@Summary		Update a task
//	@Description	Owner/admin can edit any team task; a task creator can edit their own task; an assignee can change only status. Version checking prevents lost updates and every change is recorded in history.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Task ID"	example(10)
//	@Param			request	body		dto.UpdateTaskRequest	true	"Task update payload"
//	@Success		200		{object}	dto.TaskResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	dto.ErrorResponse	"Insufficient permissions"
//	@Failure		404		{object}	dto.ErrorResponse	"Task not found"
//	@Failure		409		{object}	dto.ErrorResponse	"Version conflict"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error or no changes"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks/{id} [put]
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
		Title:       shared.FromPtr(req.Title),
		Description: shared.FromPtr(req.Description),
		Status:      shared.FromPtr(req.Status),
		AssigneeID:  shared.FromPtr(req.AssigneeID),
		Version:     shared.FromPtr(req.Version),
	})
	if err != nil {
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"task updated",
		zap.Int64("task_id", task.ID),
		zap.Int64("team_id", task.TeamID),
		zap.Uint("version", task.Version),
	)
	c.JSON(http.StatusOK, dto.NewTaskResponse(task))
}

// History handles GET /api/v1/tasks/:id/history.
//
//	@Summary		Get task history
//	@Description	Returns the ordered change log. Only members of the task team can access it.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"Task ID"	example(10)
//	@Success		200	{array}		dto.HistoryEntryResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401	{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403	{object}	dto.ErrorResponse	"User is not a team member"
//	@Failure		404	{object}	dto.ErrorResponse	"Task not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks/{id}/history [get]
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
//
//	@Summary		Add a task comment
//	@Description	Adds a comment to a task. Only members of the task team can comment.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"Task ID"	example(10)
//	@Param			request	body		dto.CreateCommentRequest	true	"Comment payload"
//	@Success		201		{object}	dto.CommentResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	dto.ErrorResponse	"User is not a team member"
//	@Failure		404		{object}	dto.ErrorResponse	"Task not found"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks/{id}/comments [post]
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

	sharedlogging.FromContext(c.Request.Context()).Info(
		"task comment added",
		zap.Int64("task_id", comment.TaskID),
		zap.Int64("comment_id", comment.ID),
	)
	c.JSON(http.StatusCreated, dto.NewCommentResponse(comment))
}

// ListComments handles GET /api/v1/tasks/:id/comments.
//
//	@Summary		List task comments
//	@Description	Returns task comments in creation order. Only members of the task team can access them.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"Task ID"	example(10)
//	@Success		200	{array}		dto.CommentResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401	{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403	{object}	dto.ErrorResponse	"User is not a team member"
//	@Failure		404	{object}	dto.ErrorResponse	"Task not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal error"
//	@Router			/tasks/{id}/comments [get]
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
