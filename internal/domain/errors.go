package domain

import "fmt"

// DomainError represents a domain-level error with a code and message.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// ErrTeamNotFound is returned when a team does not exist.
func ErrTeamNotFound(id int64) *DomainError {
	return &DomainError{
		Code:    "team_not_found",
		Message: fmt.Sprintf("team %d not found", id),
	}
}

// ErrTaskNotFound is returned when a task does not exist.
func ErrTaskNotFound(id int64) *DomainError {
	return &DomainError{
		Code:    "task_not_found",
		Message: fmt.Sprintf("task %d not found", id),
	}
}

// ErrUserNotFound is returned when a user does not exist.
func ErrUserNotFound(id int64) *DomainError {
	return &DomainError{
		Code:    "user_not_found",
		Message: fmt.Sprintf("user %d not found", id),
	}
}
