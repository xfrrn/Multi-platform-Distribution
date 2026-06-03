package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

type ReleaseService struct {
	repo Repository
}

type CreateReleaseInput struct {
	AppID          uuid.UUID `json:"-"`
	Version        string    `json:"version"`
	Channel        string    `json:"channel"`
	Changelog      string    `json:"changelog"`
	IsForced       bool      `json:"is_forced"`
	StagingPercent int       `json:"staging_percent"`
	PublishedAt    time.Time `json:"published_at"`
}

func NewReleaseService(repo Repository) *ReleaseService {
	return &ReleaseService{repo: repo}
}

func (s *ReleaseService) Create(ctx context.Context, input CreateReleaseInput) (domain.Release, error) {
	version := strings.TrimSpace(input.Version)
	if version == "" {
		return domain.Release{}, errors.New("version is required")
	}
	if input.StagingPercent < 0 || input.StagingPercent > 100 {
		return domain.Release{}, errors.New("staging_percent must be between 0 and 100")
	}
	if _, err := s.repo.GetAppByID(ctx, input.AppID); err != nil {
		return domain.Release{}, err
	}

	publishedAt := input.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = time.Now().UTC()
	}

	release := domain.Release{
		ID:             uuid.New(),
		AppID:          input.AppID,
		Version:        version,
		Channel:        normalizeChannel(input.Channel),
		Changelog:      strings.TrimSpace(input.Changelog),
		IsForced:       input.IsForced,
		StagingPercent: input.StagingPercent,
		PublishedAt:    publishedAt,
	}
	if err := s.repo.CreateRelease(ctx, &release); err != nil {
		return domain.Release{}, err
	}
	return release, nil
}

func (s *ReleaseService) ListByApp(ctx context.Context, appID uuid.UUID) ([]domain.Release, error) {
	if _, err := s.repo.GetAppByID(ctx, appID); err != nil {
		return nil, err
	}
	return s.repo.ListReleasesByApp(ctx, appID)
}
