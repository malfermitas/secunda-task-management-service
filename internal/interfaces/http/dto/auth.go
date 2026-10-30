package dto

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
}

// LoginRequest is the payload for authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse contains a JWT and the authenticated user.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
