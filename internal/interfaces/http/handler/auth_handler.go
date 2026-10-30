package handler

import (
	"net/http"

	"secunda-task-management-service/internal/interfaces/http/dto"
	userapp "secunda-task-management-service/internal/user/app"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles registration and login endpoints.
type AuthHandler struct {
	svc userapp.AuthService
}

// Register handles POST /api/v1/register.
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

	c.JSON(http.StatusCreated, dto.NewUserResponse(user))
}

// Login handles POST /api/v1/login.
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
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{
		Token: result.Token,
		User:  dto.NewUserResponse(result.User),
	})
}
