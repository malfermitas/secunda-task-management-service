package domain

import "context"

// Repository is the persistence port for the team aggregate.
type Repository interface {
	// Create persists a team and its owner membership atomically, setting team.ID.
	Create(ctx context.Context, team *Team, ownerRole Role) error
	ListByUser(ctx context.Context, userID int64) ([]Team, error)
	// Membership returns (nil, nil) when the user is not a member.
	Membership(ctx context.Context, teamID, userID int64) (*Member, error)
	AddMember(ctx context.Context, member *Member) error
}
