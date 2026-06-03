package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

type AppService struct {
	repo Repository
}

type CreateAppInput struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	IconURL        string `json:"icon_url"`
	DefaultChannel string `json:"default_channel"`
}

func NewAppService(repo Repository) *AppService {
	return &AppService{repo: repo}
}

func (s *AppService) Create(ctx context.Context, input CreateAppInput) (domain.App, error) {
	name := strings.TrimSpace(input.Name)
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	channel := normalizeChannel(input.DefaultChannel)

	if name == "" {
		return domain.App{}, errors.New("name is required")
	}
	if !slugPattern.MatchString(slug) {
		return domain.App{}, errors.New("slug must be 3-64 chars, lowercase letters, numbers, or hyphens")
	}

	app := domain.App{
		ID:             uuid.New(),
		Name:           name,
		Slug:           slug,
		Description:    strings.TrimSpace(input.Description),
		IconURL:        strings.TrimSpace(input.IconURL),
		DefaultChannel: channel,
	}
	if err := s.repo.CreateApp(ctx, &app); err != nil {
		return domain.App{}, err
	}
	return app, nil
}

func (s *AppService) List(ctx context.Context) ([]domain.App, error) {
	return s.repo.ListApps(ctx)
}

func (s *AppService) Get(ctx context.Context, id uuid.UUID) (domain.App, error) {
	return s.repo.GetAppByID(ctx, id)
}

func normalizeChannel(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		return "stable"
	}
	return channel
}
