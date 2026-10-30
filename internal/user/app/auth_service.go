package app

import (
	"context"
	"time"

	"secunda-task-management-service/internal/shared/errs"
	userdomain "secunda-task-management-service/internal/user/domain"
)

// RegisterInput contains data required to register a new user.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// LoginInput contains credentials for authentication.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult contains an issued token and the authenticated user.
type LoginResult struct {
	Token string
	User  *userdomain.User
}

// AuthService exposes authentication use cases.
type AuthService interface {
	Register(ctx context.Context, in RegisterInput) (*userdomain.User, error)
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
}

type authService struct {
	users  userdomain.Repository
	hasher PasswordHasher
	tokens TokenIssuer
}

// NewAuthService creates the authentication application service.
func NewAuthService(users userdomain.Repository, hasher PasswordHasher, tokens TokenIssuer) AuthService {
	return &authService{users: users, hasher: hasher, tokens: tokens}
}

func (s *authService) Register(ctx context.Context, in RegisterInput) (*userdomain.User, error) {
	email, err := userdomain.NewEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if len(in.Password) < 8 {
		return nil, errs.Validation("password must be at least 8 characters")
	}
	if len(in.Password) > 72 {
		return nil, errs.Validation("password must be at most 72 characters")
	}

	passwordHash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	user, err := userdomain.NewUser(email, passwordHash, in.Name, time.Now())
	if err != nil {
		return nil, err
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	email, err := userdomain.NewEmail(in.Email)
	if err != nil || in.Password == "" {
		return nil, errs.InvalidCredentials()
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errs.InvalidCredentials()
	}

	if err := s.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return nil, errs.InvalidCredentials()
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: user}, nil
}
