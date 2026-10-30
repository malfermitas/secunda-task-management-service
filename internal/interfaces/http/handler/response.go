package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"secunda-task-management-service/internal/interfaces/http/dto"
	"secunda-task-management-service/internal/interfaces/http/middleware"
	"secunda-task-management-service/internal/shared/errs"

	"github.com/gin-gonic/gin"
)

func respondError(c *gin.Context, err error) {
	var appErr *errs.Error
	if errors.As(err, &appErr) {
		respondStatus(c, statusForKind(appErr.Kind), appErr.Code, appErr.Message)
		return
	}

	slog.ErrorContext(c.Request.Context(), "request failed", "error", err)
	respondStatus(c, http.StatusInternalServerError, "internal_error", "internal server error")
}

func respondStatus(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.ErrorResponse{
		Error: dto.ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func respondBindingError(c *gin.Context, err error) {
	respondStatus(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
}

func statusForKind(kind errs.Kind) int {
	switch kind {
	case errs.KindValidation:
		return http.StatusUnprocessableEntity
	case errs.KindUnauthorized:
		return http.StatusUnauthorized
	case errs.KindForbidden:
		return http.StatusForbidden
	case errs.KindNotFound:
		return http.StatusNotFound
	case errs.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func pathID(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		respondStatus(
			c,
			http.StatusBadRequest,
			"invalid_path_parameter",
			fmt.Sprintf("%s must be a positive integer", name),
		)
		return 0, false
	}
	return value, true
}

func currentUserID(c *gin.Context) (int64, bool) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondStatus(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return 0, false
	}
	return userID, true
}
