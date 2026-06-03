package service

import (
	"context"
	"io"

	"multi-platform-distribution/internal/domain"
	"multi-platform-distribution/internal/storage"

	"github.com/google/uuid"
)

type Repository interface {
	CreateApp(ctx context.Context, app *domain.App) error
	ListApps(ctx context.Context) ([]domain.App, error)
	GetAppByID(ctx context.Context, id uuid.UUID) (domain.App, error)
	GetAppBySlug(ctx context.Context, slug string) (domain.App, error)

	CreateRelease(ctx context.Context, release *domain.Release) error
	GetReleaseByID(ctx context.Context, id uuid.UUID) (domain.Release, error)
	ListReleasesByApp(ctx context.Context, appID uuid.UUID) ([]domain.Release, error)
	GetLatestRelease(ctx context.Context, appID uuid.UUID, channel string) (domain.Release, error)

	CreateArtifact(ctx context.Context, artifact *domain.Artifact) error
	GetArtifactByID(ctx context.Context, id uuid.UUID) (domain.Artifact, error)
	ListArtifactsByRelease(ctx context.Context, releaseID uuid.UUID) ([]domain.Artifact, error)

	CreateAdminUser(ctx context.Context, user *domain.AdminUser) error
	GetAdminUserByEmail(ctx context.Context, email string) (domain.AdminUser, error)
	CountAdminUsers(ctx context.Context) (int, error)
}

type ObjectStorage interface {
	Save(ctx context.Context, key string, body io.Reader) (storage.Object, error)
	PublicPath() string
}
