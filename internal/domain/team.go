package domain

import "time"

// TeamMemberRole represents a user's role within a team.
type TeamMemberRole string

const (
	RoleOwner  TeamMemberRole = "owner"
	RoleAdmin  TeamMemberRole = "admin"
	RoleMember TeamMemberRole = "member"
)

// TeamMember represents a user's membership within a team.
type TeamMember struct {
	TeamID   int64          `json:"team_id"`
	UserID   int64          `json:"user_id"`
	Role     TeamMemberRole `json:"role"`
	JoinedAt time.Time      `json:"joined_at"`
}

// Team represents a team of users.
type Team struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	CreatedBy int64        `json:"created_by"`
	Members   []TeamMember `json:"members,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}
