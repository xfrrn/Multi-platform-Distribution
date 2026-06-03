package postgres

import (
	"context"
	"errors"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateApp(ctx context.Context, app *domain.App) error {
	row := r.pool.QueryRow(ctx, `
		insert into apps (id, name, slug, description, icon_url, default_channel)
		values ($1, $2, $3, $4, $5, $6)
		returning created_at
	`, app.ID, app.Name, app.Slug, app.Description, app.IconURL, app.DefaultChannel)
	return row.Scan(&app.CreatedAt)
}

func (r *Repository) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, err := r.pool.Query(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at
		from apps
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		var app domain.App
		if err := rows.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.IconURL, &app.DefaultChannel, &app.CreatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *Repository) GetAppByID(ctx context.Context, id uuid.UUID) (domain.App, error) {
	return r.scanApp(r.pool.QueryRow(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at
		from apps
		where id = $1
	`, id))
}

func (r *Repository) GetAppBySlug(ctx context.Context, slug string) (domain.App, error) {
	return r.scanApp(r.pool.QueryRow(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at
		from apps
		where slug = $1
	`, slug))
}

func (r *Repository) scanApp(row pgx.Row) (domain.App, error) {
	var app domain.App
	err := row.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.IconURL, &app.DefaultChannel, &app.CreatedAt)
	return app, normalizeNotFound(err)
}

func (r *Repository) CreateRelease(ctx context.Context, release *domain.Release) error {
	row := r.pool.QueryRow(ctx, `
		insert into releases (id, app_id, version, channel, changelog, is_forced, staging_percent, published_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning created_at
	`, release.ID, release.AppID, release.Version, release.Channel, release.Changelog, release.IsForced, release.StagingPercent, release.PublishedAt)
	return row.Scan(&release.CreatedAt)
}

func (r *Repository) GetReleaseByID(ctx context.Context, id uuid.UUID) (domain.Release, error) {
	var release domain.Release
	err := r.pool.QueryRow(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at
		from releases
		where id = $1
	`, id).Scan(&release.ID, &release.AppID, &release.Version, &release.Channel, &release.Changelog, &release.IsForced, &release.StagingPercent, &release.PublishedAt, &release.CreatedAt)
	return release, normalizeNotFound(err)
}

func (r *Repository) ListReleasesByApp(ctx context.Context, appID uuid.UUID) ([]domain.Release, error) {
	rows, err := r.pool.Query(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at
		from releases
		where app_id = $1
		order by published_at desc, created_at desc
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var releases []domain.Release
	for rows.Next() {
		var release domain.Release
		if err := rows.Scan(&release.ID, &release.AppID, &release.Version, &release.Channel, &release.Changelog, &release.IsForced, &release.StagingPercent, &release.PublishedAt, &release.CreatedAt); err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, rows.Err()
}

func (r *Repository) GetLatestRelease(ctx context.Context, appID uuid.UUID, channel string) (domain.Release, error) {
	var release domain.Release
	err := r.pool.QueryRow(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at
		from releases
		where app_id = $1 and channel = $2 and published_at <= now()
		order by published_at desc, created_at desc
		limit 1
	`, appID, channel).Scan(&release.ID, &release.AppID, &release.Version, &release.Channel, &release.Changelog, &release.IsForced, &release.StagingPercent, &release.PublishedAt, &release.CreatedAt)
	return release, normalizeNotFound(err)
}

func (r *Repository) CreateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	row := r.pool.QueryRow(ctx, `
		insert into artifacts (id, release_id, platform, arch, file_type, file_url, storage_key, file_size, sha512)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning created_at
	`, artifact.ID, artifact.ReleaseID, artifact.Platform, artifact.Arch, artifact.FileType, artifact.FileURL, artifact.StorageKey, artifact.FileSize, artifact.SHA512)
	return row.Scan(&artifact.CreatedAt)
}

func (r *Repository) GetArtifactByID(ctx context.Context, id uuid.UUID) (domain.Artifact, error) {
	var artifact domain.Artifact
	err := r.pool.QueryRow(ctx, `
		select id, release_id, platform, arch, file_type, file_url, storage_key, file_size, sha512, created_at
		from artifacts
		where id = $1
	`, id).Scan(&artifact.ID, &artifact.ReleaseID, &artifact.Platform, &artifact.Arch, &artifact.FileType, &artifact.FileURL, &artifact.StorageKey, &artifact.FileSize, &artifact.SHA512, &artifact.CreatedAt)
	return artifact, normalizeNotFound(err)
}

func (r *Repository) ListArtifactsByRelease(ctx context.Context, releaseID uuid.UUID) ([]domain.Artifact, error) {
	rows, err := r.pool.Query(ctx, `
		select id, release_id, platform, arch, file_type, file_url, storage_key, file_size, sha512, created_at
		from artifacts
		where release_id = $1
		order by platform, arch, file_type
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []domain.Artifact
	for rows.Next() {
		var artifact domain.Artifact
		if err := rows.Scan(&artifact.ID, &artifact.ReleaseID, &artifact.Platform, &artifact.Arch, &artifact.FileType, &artifact.FileURL, &artifact.StorageKey, &artifact.FileSize, &artifact.SHA512, &artifact.CreatedAt); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func (r *Repository) CreateAdminUser(ctx context.Context, user *domain.AdminUser) error {
	row := r.pool.QueryRow(ctx, `
		insert into admin_users (id, email, name, password_hash, is_active)
		values ($1, lower($2), $3, $4, $5)
		returning created_at
	`, user.ID, user.Email, user.Name, user.PasswordHash, user.IsActive)
	return row.Scan(&user.CreatedAt)
}

func (r *Repository) GetAdminUserByEmail(ctx context.Context, email string) (domain.AdminUser, error) {
	var user domain.AdminUser
	err := r.pool.QueryRow(ctx, `
		select id, email, name, password_hash, is_active, created_at
		from admin_users
		where email = lower($1)
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.IsActive, &user.CreatedAt)
	return user, normalizeNotFound(err)
}

func (r *Repository) CountAdminUsers(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `select count(*) from admin_users`).Scan(&count)
	return count, err
}

func normalizeNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
