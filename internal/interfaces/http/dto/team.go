package dto

// CreateTeamRequest is the payload for creating a team.
type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255" example:"Platform Team"`
}

// InviteRequest is the payload for inviting a user to a team.
type InviteRequest struct {
	UserID int64  `json:"user_id" binding:"required,gt=0" example:"2"`
	Role   string `json:"role" binding:"required,oneof=admin member" enums:"admin,member" example:"member"`
}
