package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"multi-platform-distribution/internal/auth"
	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type AuthService struct {
	repo   Repository
	tokens *auth.TokenManager
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	AccessToken string           `json:"access_token"`
	TokenType   string           `json:"token_type"`
	ExpiresAt   time.Time        `json:"expires_at"`
	Admin       domain.AdminUser `json:"admin"`
}

type BootstrapAdminInput struct {
	Email    string
	Name     string
	Password string
}

func NewAuthService(repo Repository, tokens *auth.TokenManager) *AuthService {
	return &AuthService{repo: repo, tokens: tokens}
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	admin, err := s.repo.GetAdminUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}
	if !admin.IsActive {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, expiresAt, err := s.tokens.Issue(admin.ID, admin.Email)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		Admin:       admin,
	}, nil
}

func (s *AuthService) BootstrapDefaultAdmin(ctx context.Context, input BootstrapAdminInput) error {
	email := normalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return nil
	}

	count, err := s.repo.CountAdminUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := domain.AdminUser{
		ID:           uuid.New(),
		Email:        email,
		Name:         strings.TrimSpace(input.Name),
		PasswordHash: string(passwordHash),
		IsActive:     true,
	}
	return s.repo.CreateAdminUser(ctx, &admin)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
