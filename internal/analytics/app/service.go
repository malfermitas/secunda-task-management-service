package app

import (
	"context"

	analyticsdomain "secunda-task-management-service/internal/analytics/domain"
	"secunda-task-management-service/internal/shared/errs"
	teamdomain "secunda-task-management-service/internal/team/domain"
)

// Service exposes analytics use cases.
type Service interface {
	TeamStats(ctx context.Context, actorID, teamID int64) (*analyticsdomain.TeamStats, error)
}

type service struct {
	teams teamdomain.Repository
	stats analyticsdomain.Repository
}

// NewService creates the analytics application service.
func NewService(teams teamdomain.Repository, stats analyticsdomain.Repository) Service {
	return &service{teams: teams, stats: stats}
}

func (s *service) TeamStats(ctx context.Context, actorID, teamID int64) (*analyticsdomain.TeamStats, error) {
	membership, err := s.teams.Membership(ctx, teamID, actorID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, teamdomain.ErrNotMember(teamID)
	}
	if !membership.Role.CanViewAnalytics() {
		return nil, errs.Forbidden("only team owner or admin can view analytics")
	}

	return s.stats.TeamStats(ctx, teamID)
}
