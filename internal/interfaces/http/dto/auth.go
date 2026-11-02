package dto

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"jane@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"password123"`
	Name     string `json:"name" binding:"required,min=1,max=255" example:"Jane Doe"`
}

// LoginRequest is the payload for authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"jane@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse contains a JWT and the authenticated user.
type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}
