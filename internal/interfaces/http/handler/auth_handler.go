package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"secunda-task-management-service/internal/interfaces/http/dto"
	sharedlogging "secunda-task-management-service/internal/shared/logging"
	userapp "secunda-task-management-service/internal/user/app"
)

// AuthHandler handles registration and login endpoints.
type AuthHandler struct {
	svc userapp.AuthService
}

// Register handles POST /api/v1/register.
//
//	@Summary		Register a user
//	@Description	Creates a user account. The email must be unique.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	dto.UserResponse
//	@Failure		409		{object}	dto.ErrorResponse	"Email already registered"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	user, err := h.svc.Register(c.Request.Context(), userapp.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"user registered",
		zap.Int64("user_id", user.ID),
	)
	c.JSON(http.StatusCreated, dto.NewUserResponse(user))
}

// Login handles POST /api/v1/login.
//
//	@Summary		Authenticate and get a JWT
//	@Description	Verifies credentials and returns a bearer token plus the authenticated user.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Login payload"
//	@Success		200		{object}	dto.LoginResponse
//	@Failure		401		{object}	dto.ErrorResponse	"Invalid email or password"
//	@Failure		422		{object}	dto.ErrorResponse	"Validation error"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal error"
//	@Router			/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	result, err := h.svc.Login(c.Request.Context(), userapp.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		sharedlogging.FromContext(c.Request.Context()).Warn("login failed")
		respondError(c, err)
		return
	}

	sharedlogging.FromContext(c.Request.Context()).Info(
		"user logged in",
		zap.Int64("user_id", result.User.ID),
	)
	c.JSON(http.StatusOK, dto.LoginResponse{
		Token: result.Token,
		User:  dto.NewUserResponse(result.User),
	})
}
