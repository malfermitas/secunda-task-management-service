package handler

import (
	"net/http"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/interfaces/http/dto"
	sharedlogging "secunda-task-management-service/internal/shared/logging"
	teamapp "secunda-task-management-service/internal/team/app"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TeamsHandler handles team and analytics endpoints.
type TeamsHandler struct {
	teams teamapp.Service
	stats analyticsapp.Service
}

// Create handles POST /api/v1/teams.
//
//	@Summary		Create a team
//	@Description	Creates a team; the current user becomes its owner.
//	@Tags			Teams
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateTeamRequest	true	"Team payload"
//	@Success		201		{object}	dto.TeamResponse
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/teams [post]
func (h *TeamsHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	team, err := h.teams.Create(c.Request.Context(), userID, req.Name)
	if err != nil {
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"team created",
		zap.Int64("team_id", team.ID),
	)
	c.JSON(http.StatusCreated, dto.NewTeamResponse(*team))
}

// List handles GET /api/v1/teams.
//
//	@Summary		List current user teams
//	@Description	Returns only teams where the current user is a member or creator.
//	@Tags			Teams
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		dto.TeamResponse
//	@Failure		401	{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal error"
//	@Router			/teams [get]
func (h *TeamsHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	teams, err := h.teams.List(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewTeamResponses(teams))
}

// Invite handles POST /api/v1/teams/:id/invite.
//
//	@Summary		Invite a user to a team
//	@Description	Owner or admin only. The owner role cannot be granted through an invite.
//	@Tags			Teams
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int					true	"Team ID"	example(1)
//	@Param			request	body	dto.InviteRequest	true	"Invite payload"
//	@Success		204		"Member invited"
//	@Failure		400		{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	dto.ErrorResponse	"Only owner or admin can invite"
//	@Failure		404		{object}	dto.ErrorResponse	"Invitee user not found"
//	@Failure		409		{object}	dto.ErrorResponse	"User is already a member"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/teams/{id}/invite [post]
func (h *TeamsHandler) Invite(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	teamID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	err := h.teams.Invite(c.Request.Context(), userID, teamapp.InviteInput{
		TeamID: teamID,
		UserID: req.UserID,
		Role:   req.Role,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"team member invited",
		zap.Int64("team_id", teamID),
		zap.Int64("invitee_id", req.UserID),
		zap.String("role", req.Role),
	)
	c.Status(http.StatusNoContent)
}

// Stats handles GET /api/v1/teams/:team_id/stats.
//
//	@Summary		Get team analytics
//	@Description	Owner or admin only. Returns task counts by status, top-3 assignees by closed tasks in the last 30 days, average close time and comments per task.
//	@Tags			Analytics
//	@Security		BearerAuth
//	@Produce		json
//	@Param			team_id	path		int	true	"Team ID"	example(1)
//	@Success		200		{object}	dto.TeamStatsResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Invalid path parameter"
//	@Failure		401		{object}	dto.ErrorResponse	"Authentication required"
//	@Failure		403		{object}	dto.ErrorResponse	"Only owner/admin can view analytics or user is not a team member"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/teams/{team_id}/stats [get]
func (h *TeamsHandler) Stats(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	teamID, ok := pathID(c, "team_id")
	if !ok {
		return
	}

	stats, err := h.stats.TeamStats(c.Request.Context(), userID, teamID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewTeamStatsResponse(stats))
}
