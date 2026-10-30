package dto

// CreateTeamRequest is the payload for creating a team.
type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

// InviteRequest is the payload for inviting a user to a team.
type InviteRequest struct {
	UserID int64  `json:"user_id" binding:"required,gt=0"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}
