package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		returning created_at, updated_at, archived_at
	`, app.ID, app.Name, app.Slug, app.Description, app.IconURL, app.DefaultChannel)
	return row.Scan(&app.CreatedAt, &app.UpdatedAt, &app.ArchivedAt)
}

func (r *Repository) UpdateApp(ctx context.Context, app *domain.App) error {
	row := r.pool.QueryRow(ctx, `
		update apps
		set name = $2,
			description = $3,
			icon_url = $4,
			default_channel = $5,
			updated_at = now()
		where id = $1 and archived_at is null
		returning id, name, slug, description, icon_url, default_channel, created_at, updated_at, archived_at
	`, app.ID, app.Name, app.Description, app.IconURL, app.DefaultChannel)
	updated, err := r.scanApp(row)
	if err != nil {
		return err
	}
	*app = updated
	return nil
}

func (r *Repository) ArchiveApp(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		update apps
		set archived_at = coalesce(archived_at, now()),
			updated_at = now()
		where id = $1 and archived_at is null
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, err := r.pool.Query(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at, updated_at, archived_at
		from apps
		where archived_at is null
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		var app domain.App
		if err := rows.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.IconURL, &app.DefaultChannel, &app.CreatedAt, &app.UpdatedAt, &app.ArchivedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *Repository) GetAppByID(ctx context.Context, id uuid.UUID) (domain.App, error) {
	return r.scanApp(r.pool.QueryRow(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at, updated_at, archived_at
		from apps
		where id = $1 and archived_at is null
	`, id))
}

func (r *Repository) GetAppBySlug(ctx context.Context, slug string) (domain.App, error) {
	return r.scanApp(r.pool.QueryRow(ctx, `
		select id, name, slug, description, icon_url, default_channel, created_at, updated_at, archived_at
		from apps
		where slug = $1 and archived_at is null
	`, slug))
}

func (r *Repository) scanApp(row pgx.Row) (domain.App, error) {
	var app domain.App
	err := row.Scan(&app.ID, &app.Name, &app.Slug, &app.Description, &app.IconURL, &app.DefaultChannel, &app.CreatedAt, &app.UpdatedAt, &app.ArchivedAt)
	return app, normalizeNotFound(err)
}

func (r *Repository) CreateRelease(ctx context.Context, release *domain.Release) error {
	row := r.pool.QueryRow(ctx, `
		insert into releases (id, app_id, version, channel, changelog, is_forced, staging_percent, published_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		returning created_at, updated_at, archived_at
	`, release.ID, release.AppID, release.Version, release.Channel, release.Changelog, release.IsForced, release.StagingPercent, release.PublishedAt)
	return row.Scan(&release.CreatedAt, &release.UpdatedAt, &release.ArchivedAt)
}

func (r *Repository) UpdateRelease(ctx context.Context, release *domain.Release) error {
	row := r.pool.QueryRow(ctx, `
		update releases
		set version = $2,
			channel = $3,
			changelog = $4,
			is_forced = $5,
			staging_percent = $6,
			published_at = $7,
			updated_at = now()
		where id = $1 and archived_at is null
		returning id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at, updated_at, archived_at
	`, release.ID, release.Version, release.Channel, release.Changelog, release.IsForced, release.StagingPercent, release.PublishedAt)
	updated, err := scanRelease(row)
	if err != nil {
		return err
	}
	*release = updated
	return nil
}

func (r *Repository) ArchiveRelease(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		update releases
		set archived_at = coalesce(archived_at, now()),
			updated_at = now()
		where id = $1 and archived_at is null
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) GetReleaseByID(ctx context.Context, id uuid.UUID) (domain.Release, error) {
	release, err := scanRelease(r.pool.QueryRow(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at, updated_at, archived_at
		from releases
		where id = $1 and archived_at is null
	`, id))
	return release, err
}

func (r *Repository) ListReleasesByApp(ctx context.Context, appID uuid.UUID) ([]domain.Release, error) {
	rows, err := r.pool.Query(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at, updated_at, archived_at
		from releases
		where app_id = $1 and archived_at is null
		order by published_at desc, created_at desc
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var releases []domain.Release
	for rows.Next() {
		release, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, rows.Err()
}

func (r *Repository) GetLatestRelease(ctx context.Context, appID uuid.UUID, channel string) (domain.Release, error) {
	release, err := scanRelease(r.pool.QueryRow(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at, updated_at, archived_at
		from releases
		where app_id = $1 and channel = $2 and published_at <= now() and archived_at is null
		order by published_at desc, created_at desc
		limit 1
	`, appID, channel))
	return release, err
}

func (r *Repository) ListPublishedReleasesByAppChannel(ctx context.Context, appID uuid.UUID, channel string) ([]domain.Release, error) {
	rows, err := r.pool.Query(ctx, `
		select id, app_id, version, channel, changelog, is_forced, staging_percent, published_at, created_at, updated_at, archived_at
		from releases
		where app_id = $1 and channel = $2 and published_at <= now() and archived_at is null
		order by published_at desc, created_at desc
	`, appID, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var releases []domain.Release
	for rows.Next() {
		release, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, rows.Err()
}

func (r *Repository) CreateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	artifact.SourceType = normalizeArtifactSource(artifact.SourceType)
	row := r.pool.QueryRow(ctx, `
		insert into artifacts (
			id, release_id, platform, arch, file_type, file_name, file_url, storage_key,
			file_size, sha512, source_type, anyshare_docid, anyshare_rev, anyshare_name
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		returning created_at, updated_at, archived_at
	`, artifact.ID, artifact.ReleaseID, artifact.Platform, artifact.Arch, artifact.FileType, artifact.FileName, artifact.FileURL, artifact.StorageKey, artifact.FileSize, artifact.SHA512, artifact.SourceType, artifact.AnyshareDocID, artifact.AnyshareRev, artifact.AnyshareName)
	return row.Scan(&artifact.CreatedAt, &artifact.UpdatedAt, &artifact.ArchivedAt)
}

func (r *Repository) UpdateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	row := r.pool.QueryRow(ctx, `
		update artifacts
		set platform = $2,
			arch = $3,
			file_type = $4,
			file_name = $5,
			updated_at = now()
		where id = $1 and archived_at is null
		returning id, release_id, platform, arch, file_type, file_name, file_url, storage_key, file_size, sha512,
			source_type, anyshare_docid, anyshare_rev, anyshare_name, created_at, updated_at, archived_at
	`, artifact.ID, artifact.Platform, artifact.Arch, artifact.FileType, artifact.FileName)
	updated, err := scanArtifact(row)
	if err != nil {
		return err
	}
	*artifact = updated
	return nil
}

func (r *Repository) UpdateArtifactFile(ctx context.Context, artifact *domain.Artifact) error {
	row := r.pool.QueryRow(ctx, `
		update artifacts
		set file_name = $2,
			file_url = $3,
			storage_key = $4,
			file_size = $5,
			sha512 = $6,
			source_type = $7,
			anyshare_docid = $8,
			anyshare_rev = $9,
			anyshare_name = $10,
			updated_at = now()
		where id = $1 and archived_at is null
		returning id, release_id, platform, arch, file_type, file_name, file_url, storage_key, file_size, sha512,
			source_type, anyshare_docid, anyshare_rev, anyshare_name, created_at, updated_at, archived_at
	`, artifact.ID, artifact.FileName, artifact.FileURL, artifact.StorageKey, artifact.FileSize, artifact.SHA512, normalizeArtifactSource(artifact.SourceType), artifact.AnyshareDocID, artifact.AnyshareRev, artifact.AnyshareName)
	updated, err := scanArtifact(row)
	if err != nil {
		return err
	}
	*artifact = updated
	return nil
}

func (r *Repository) ArchiveArtifact(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		update artifacts
		set archived_at = coalesce(archived_at, now()),
			updated_at = now()
		where id = $1 and archived_at is null
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) GetArtifactByID(ctx context.Context, id uuid.UUID) (domain.Artifact, error) {
	artifact, err := scanArtifact(r.pool.QueryRow(ctx, `
		select id, release_id, platform, arch, file_type, file_name, file_url, storage_key, file_size, sha512,
			source_type, anyshare_docid, anyshare_rev, anyshare_name, created_at, updated_at, archived_at
		from artifacts
		where id = $1 and archived_at is null
	`, id))
	return artifact, err
}

func (r *Repository) ListArtifactsByRelease(ctx context.Context, releaseID uuid.UUID) ([]domain.Artifact, error) {
	rows, err := r.pool.Query(ctx, `
		select id, release_id, platform, arch, file_type, file_name, file_url, storage_key, file_size, sha512,
			source_type, anyshare_docid, anyshare_rev, anyshare_name, created_at, updated_at, archived_at
		from artifacts
		where release_id = $1 and archived_at is null
		order by platform, arch, file_type
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []domain.Artifact
	for rows.Next() {
		artifact, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func scanRelease(row pgx.Row) (domain.Release, error) {
	var release domain.Release
	err := row.Scan(&release.ID, &release.AppID, &release.Version, &release.Channel, &release.Changelog, &release.IsForced, &release.StagingPercent, &release.PublishedAt, &release.CreatedAt, &release.UpdatedAt, &release.ArchivedAt)
	return release, normalizeNotFound(err)
}

func scanArtifact(row pgx.Row) (domain.Artifact, error) {
	var artifact domain.Artifact
	err := row.Scan(&artifact.ID, &artifact.ReleaseID, &artifact.Platform, &artifact.Arch, &artifact.FileType, &artifact.FileName, &artifact.FileURL, &artifact.StorageKey, &artifact.FileSize, &artifact.SHA512, &artifact.SourceType, &artifact.AnyshareDocID, &artifact.AnyshareRev, &artifact.AnyshareName, &artifact.CreatedAt, &artifact.UpdatedAt, &artifact.ArchivedAt)
	if artifact.SourceType == "" {
		artifact.SourceType = "managed"
	}
	return artifact, normalizeNotFound(err)
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

func (r *Repository) CreateUpdateRequestEvent(ctx context.Context, event *domain.UpdateRequestEvent) error {
	row := r.pool.QueryRow(ctx, `
		insert into update_requests (
			id, app_id, app_slug, release_id, version, channel, platform, arch,
			client_id, format, matched, staged_hit, ip, user_agent
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		returning created_at
	`, event.ID, event.AppID, event.AppSlug, event.ReleaseID, event.Version, event.Channel, event.Platform, event.Arch, event.ClientID, event.Format, event.Matched, event.StagedHit, event.IP, event.UserAgent)
	return row.Scan(&event.CreatedAt)
}

func (r *Repository) CreateDownloadEvent(ctx context.Context, event *domain.DownloadEvent) error {
	row := r.pool.QueryRow(ctx, `
		insert into download_events (
			id, app_id, release_id, artifact_id, version, platform, arch,
			file_type, file_name, client_id, ip, user_agent
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		returning created_at
	`, event.ID, event.AppID, event.ReleaseID, event.ArtifactID, event.Version, event.Platform, event.Arch, event.FileType, event.FileName, event.ClientID, event.IP, event.UserAgent)
	return row.Scan(&event.CreatedAt)
}

func (r *Repository) GetStatsSummary(ctx context.Context, filter domain.StatsFilter) (domain.StatsSummary, error) {
	var summary domain.StatsSummary
	updateWhere, args := statsWhere("u", false, filter)
	downloadWhere, _ := statsWhere("d", true, filter)
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		select
			(select count(*) from update_requests u where u.created_at >= date_trunc('day', now()) and %s),
			(select count(*) from download_events d join releases r on r.id = d.release_id where d.created_at >= date_trunc('day', now()) and %s),
			(select count(*) from download_events d join releases r on r.id = d.release_id where d.created_at >= now() - interval '7 days' and %s),
			(select count(distinct client_id) from (
				select u.client_id from update_requests u where u.client_id <> '' and u.created_at >= now() - interval '7 days' and %s
				union all
				select d.client_id from download_events d join releases r on r.id = d.release_id where d.client_id <> '' and d.created_at >= now() - interval '7 days' and %s
			) clients),
			(select count(*) from update_requests u where u.staged_hit = true and u.created_at >= now() - interval '7 days' and %s),
			(select count(*) from download_events d join releases r on r.id = d.release_id where r.is_forced = true and d.created_at >= now() - interval '7 days' and %s),
			(select count(*) from update_requests u where %s),
			(select count(*) from download_events d join releases r on r.id = d.release_id where %s)
	`, updateWhere.SQL, downloadWhere.SQL, downloadWhere.SQL, updateWhere.SQL, downloadWhere.SQL, updateWhere.SQL, downloadWhere.SQL, updateWhere.SQL, downloadWhere.SQL), args...).Scan(
		&summary.TodayUpdateRequests,
		&summary.TodayDownloads,
		&summary.Downloads7d,
		&summary.ActiveClients7d,
		&summary.StagingHits7d,
		&summary.ForcedDownloads7d,
		&summary.TotalUpdateRequests,
		&summary.TotalDownloads,
	)
	if err != nil {
		return domain.StatsSummary{}, err
	}

	summary.UpdateTrend, err = r.statsTrend(ctx, "update_requests", "u", false, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	summary.DownloadTrend, err = r.statsTrend(ctx, "download_events", "d", true, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	summary.PlatformBreakdown, err = r.statsBreakdown(ctx, "download_events", "d", "d.platform", true, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	summary.ArchBreakdown, err = r.statsBreakdown(ctx, "download_events", "d", "d.arch", true, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	summary.ChannelBreakdown, err = r.statsBreakdown(ctx, "update_requests", "u", "u.channel", false, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}
	summary.VersionBreakdown, err = r.statsBreakdown(ctx, "download_events", "d", "d.version", true, filter)
	if err != nil {
		return domain.StatsSummary{}, err
	}

	return summary, nil
}

func (r *Repository) ListUpdateRequestEvents(ctx context.Context, filter domain.StatsFilter) ([]domain.UpdateRequestEvent, error) {
	where, args := statsWhere("u", false, filter)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		select id, app_id, app_slug, release_id, version, channel, platform, arch,
			client_id, format, matched, staged_hit, ip, user_agent, created_at
		from update_requests u
		where %s
		order by created_at desc
		limit $%d offset $%d
	`, where.SQL, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.UpdateRequestEvent
	for rows.Next() {
		var event domain.UpdateRequestEvent
		if err := rows.Scan(&event.ID, &event.AppID, &event.AppSlug, &event.ReleaseID, &event.Version, &event.Channel, &event.Platform, &event.Arch, &event.ClientID, &event.Format, &event.Matched, &event.StagedHit, &event.IP, &event.UserAgent, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) ListDownloadEvents(ctx context.Context, filter domain.StatsFilter) ([]domain.DownloadEvent, error) {
	where, args := statsWhere("d", true, filter)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		select d.id, d.app_id, d.release_id, d.artifact_id, d.version, d.platform, d.arch,
			d.file_type, d.file_name, d.client_id, d.ip, d.user_agent, d.created_at
		from download_events d
		join releases r on r.id = d.release_id
		where %s
		order by d.created_at desc
		limit $%d offset $%d
	`, where.SQL, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.DownloadEvent
	for rows.Next() {
		var event domain.DownloadEvent
		if err := rows.Scan(&event.ID, &event.AppID, &event.ReleaseID, &event.ArtifactID, &event.Version, &event.Platform, &event.Arch, &event.FileType, &event.FileName, &event.ClientID, &event.IP, &event.UserAgent, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) statsTrend(ctx context.Context, table, alias string, joinRelease bool, filter domain.StatsFilter) ([]domain.StatsPoint, error) {
	where, args := statsWhere(alias, joinRelease, filter)
	join := ""
	if joinRelease {
		join = "join releases r on r.id = " + alias + ".release_id"
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		select to_char(days.day, 'YYYY-MM-DD') as date, coalesce(counts.count, 0) as count
		from generate_series(date_trunc('day', now()) - interval '6 days', date_trunc('day', now()), interval '1 day') as days(day)
		left join (
			select date_trunc('day', %s.created_at) as day, count(*) as count
			from %s %s
			%s
			where %s.created_at >= date_trunc('day', now()) - interval '6 days'
				and %s
			group by date_trunc('day', %s.created_at)
		) counts on counts.day = days.day
		order by days.day
	`, alias, table, alias, join, alias, where.SQL, alias), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]domain.StatsPoint, 0, 7)
	for rows.Next() {
		var point domain.StatsPoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (r *Repository) statsBreakdown(ctx context.Context, table, alias, column string, joinRelease bool, filter domain.StatsFilter) ([]domain.StatsBreakdown, error) {
	where, args := statsWhere(alias, joinRelease, filter)
	join := ""
	if joinRelease {
		join = "join releases r on r.id = " + alias + ".release_id"
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		select coalesce(nullif(%s, ''), 'unknown') as key, count(*) as count
		from %s %s
		%s
		where %s.created_at >= now() - interval '30 days'
			and %s
		group by coalesce(nullif(%s, ''), 'unknown')
		order by count desc, key asc
		limit 10
	`, column, table, alias, join, alias, where.SQL, column), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.StatsBreakdown, 0)
	for rows.Next() {
		var item domain.StatsBreakdown
		if err := rows.Scan(&item.Key, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type statsWhereClause struct {
	SQL  string
	Args []any
}

func statsWhere(alias string, joinRelease bool, filter domain.StatsFilter) (statsWhereClause, []any) {
	conditions := make([]string, 0, 8)
	args := make([]any, 0, 8)
	next := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.AppID != nil {
		conditions = append(conditions, fmt.Sprintf("%s.app_id = %s", alias, next(*filter.AppID)))
	}
	if filter.ReleaseID != nil {
		conditions = append(conditions, fmt.Sprintf("%s.release_id = %s", alias, next(*filter.ReleaseID)))
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("%s.created_at >= %s", alias, next(*filter.DateFrom)))
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("%s.created_at <= %s", alias, next(*filter.DateTo)))
	}
	if filter.Channel != "" {
		column := alias + ".channel"
		if joinRelease {
			column = "r.channel"
		}
		conditions = append(conditions, fmt.Sprintf("%s = %s", column, next(filter.Channel)))
	}
	if filter.Platform != "" {
		conditions = append(conditions, fmt.Sprintf("%s.platform = %s", alias, next(filter.Platform)))
	}
	if filter.Arch != "" {
		conditions = append(conditions, fmt.Sprintf("%s.arch = %s", alias, next(filter.Arch)))
	}
	if filter.ClientID != "" {
		conditions = append(conditions, fmt.Sprintf("%s.client_id = %s", alias, next(filter.ClientID)))
	}
	if len(conditions) == 0 {
		conditions = append(conditions, "true")
	}

	clause := statsWhereClause{
		SQL:  strings.Join(conditions, " and "),
		Args: args,
	}
	return clause, args
}

func normalizeNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func normalizeArtifactSource(sourceType string) string {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	if sourceType == "" {
		return "managed"
	}
	return sourceType
}
