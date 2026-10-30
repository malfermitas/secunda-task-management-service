package domain

import (
	"fmt"
	"strings"
	"time"

	"secunda-task-management-service/internal/shared/errs"
)

// Role is a value object describing a member's responsibilities in a team.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// ParseRole validates and converts a raw role string.
func ParseRole(raw string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(raw)))
	if !role.Valid() {
		return "", ErrInvalidRole(raw)
	}
	return role, nil
}

// Valid reports whether the role is supported.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// CanManageMembers reports whether the role may invite users and change roles.
func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanEditAnyTask reports whether the role may edit any task in the team.
func (r Role) CanEditAnyTask() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanViewAnalytics reports whether the role may view team analytics.
func (r Role) CanViewAnalytics() bool {
	return r == RoleOwner || r == RoleAdmin
}

// Team is the team aggregate root.
type Team struct {
	ID        int64
	Name      string
	CreatedBy int64
	CreatedAt time.Time
}

// NewTeam creates a validated team entity.
func NewTeam(name string, ownerID int64, now time.Time) (*Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errs.Validation("team name is required")
	}
	if len(name) > 255 {
		return nil, errs.Validation("team name must be at most 255 characters")
	}
	if ownerID <= 0 {
		return nil, errs.Validation("owner id is required")
	}

	return &Team{
		Name:      name,
		CreatedBy: ownerID,
		CreatedAt: now.UTC(),
	}, nil
}

// Member is the membership entity linking a user to a team.
type Member struct {
	TeamID int64
	UserID int64
	Role   Role
}

// NewMember creates a validated membership entity.
func NewMember(teamID, userID int64, role Role) (*Member, error) {
	if teamID <= 0 {
		return nil, errs.Validation("team id is required")
	}
	if userID <= 0 {
		return nil, errs.Validation("user id is required")
	}
	if !role.Valid() {
		return nil, ErrInvalidRole(string(role))
	}

	return &Member{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
	}, nil
}

// ErrNotMember is returned when a user is not a member of a team.
func ErrNotMember(teamID int64) *errs.Error {
	return errs.New(
		"not_team_member",
		fmt.Sprintf("user is not a member of team %d", teamID),
		errs.KindForbidden,
	)
}

// ErrAlreadyMember is returned when adding an existing member.
func ErrAlreadyMember(teamID, userID int64) *errs.Error {
	return errs.Conflict(
		"already_member",
		fmt.Sprintf("user %d is already a member of team %d", userID, teamID),
	)
}

// ErrInvalidRole is returned when a role is not supported.
func ErrInvalidRole(role string) *errs.Error {
	return errs.Validation(fmt.Sprintf("invalid role %q", role))
}

// ErrOwnerRoleNotAllowed is returned when an invite tries to grant ownership.
func ErrOwnerRoleNotAllowed() *errs.Error {
	return errs.New("invalid_role", "role 'owner' cannot be granted via invite", errs.KindValidation)
}
