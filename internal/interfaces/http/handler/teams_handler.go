package handler

import (
	"net/http"

	analyticsapp "secunda-task-management-service/internal/analytics/app"
	"secunda-task-management-service/internal/interfaces/http/dto"
	teamapp "secunda-task-management-service/internal/team/app"

	"github.com/gin-gonic/gin"
)

// TeamsHandler handles team and analytics endpoints.
type TeamsHandler struct {
	teams teamapp.Service
	stats analyticsapp.Service
}

// Create handles POST /api/v1/teams.
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

	c.JSON(http.StatusCreated, dto.NewTeamResponse(*team))
}

// List handles GET /api/v1/teams.
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

	c.Status(http.StatusNoContent)
}

// Stats handles GET /api/v1/teams/:team_id/stats.
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

	c.JSON(http.StatusOK, stats)
}
