package app

import (
	"context"
	"time"

	"secunda-task-management-service/internal/shared/errs"
	teamdomain "secunda-task-management-service/internal/team/domain"
	userdomain "secunda-task-management-service/internal/user/domain"
)

// InviteInput contains data required to add a user to a team.
type InviteInput struct {
	TeamID int64
	UserID int64
	Role   string
}

// Service exposes team use cases.
type Service interface {
	Create(ctx context.Context, actorID int64, name string) (*teamdomain.Team, error)
	List(ctx context.Context, actorID int64) ([]teamdomain.Team, error)
	Invite(ctx context.Context, actorID int64, in InviteInput) error
}

type service struct {
	teams teamdomain.Repository
	users userdomain.Repository
}

// NewService creates the team application service.
func NewService(teams teamdomain.Repository, users userdomain.Repository) Service {
	return &service{teams: teams, users: users}
}

func (s *service) Create(ctx context.Context, actorID int64, name string) (*teamdomain.Team, error) {
	team, err := teamdomain.NewTeam(name, actorID, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.teams.Create(ctx, team, teamdomain.RoleOwner); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *service) List(ctx context.Context, actorID int64) ([]teamdomain.Team, error) {
	teams, err := s.teams.ListByUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []teamdomain.Team{}
	}
	return teams, nil
}

func (s *service) Invite(ctx context.Context, actorID int64, in InviteInput) error {
	role, err := teamdomain.ParseRole(in.Role)
	if err != nil {
		return err
	}
	if role == teamdomain.RoleOwner {
		return teamdomain.ErrOwnerRoleNotAllowed()
	}

	membership, err := s.membership(ctx, in.TeamID, actorID)
	if err != nil {
		return err
	}
	if !membership.Role.CanManageMembers() {
		return errs.Forbidden("only team owner or admin can invite members")
	}

	invitee, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if invitee == nil {
		return userdomain.ErrNotFound(in.UserID)
	}

	member, err := teamdomain.NewMember(in.TeamID, in.UserID, role)
	if err != nil {
		return err
	}
	return s.teams.AddMember(ctx, member)
}

func (s *service) membership(ctx context.Context, teamID, userID int64) (*teamdomain.Member, error) {
	membership, err := s.teams.Membership(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, teamdomain.ErrNotMember(teamID)
	}
	return membership, nil
}
