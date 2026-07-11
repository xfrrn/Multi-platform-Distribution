package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureDatabase(ctx context.Context, databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	databaseName := config.ConnConfig.Database
	if databaseName == "" {
		return nil
	}

	conn, err := pgx.ConnectConfig(ctx, config.ConnConfig)
	if err == nil {
		return conn.Close(ctx)
	}
	if !isPgError(err, "3D000") {
		return fmt.Errorf("connect configured database: %w", err)
	}

	for _, maintenanceDB := range []string{"postgres", "template1"} {
		if maintenanceDB == databaseName {
			continue
		}
		maintenanceConfig := config.ConnConfig.Copy()
		maintenanceConfig.Database = maintenanceDB

		conn, err := pgx.ConnectConfig(ctx, maintenanceConfig)
		if err != nil {
			continue
		}

		var exists bool
		err = conn.QueryRow(ctx, `select exists(select 1 from pg_database where datname = $1)`, databaseName).Scan(&exists)
		if err == nil && !exists {
			_, err = conn.Exec(ctx, `create database `+quoteIdentifier(databaseName))
		}
		closeErr := conn.Close(ctx)
		if err == nil || isPgError(err, "42P04") {
			return closeErr
		}
	}

	return fmt.Errorf("create database %q: database is missing and no maintenance database was reachable", databaseName)
}

func EnsureSchema(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin schema initialization: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit schema initialization: %w", err)
	}
	return nil
}

func isPgError(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

const schemaSQL = `
create table if not exists apps (
	id uuid primary key,
	name varchar(160) not null,
	slug varchar(80) not null unique,
	description text not null default '',
	icon_url varchar(500) not null default '',
	default_channel varchar(40) not null default 'stable',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	archived_at timestamptz
);

create table if not exists releases (
	id uuid primary key,
	app_id uuid not null references apps(id) on delete cascade,
	version varchar(80) not null,
	channel varchar(40) not null default 'stable',
	changelog text not null default '',
	is_forced boolean not null default false,
	staging_percent integer not null default 100 check (staging_percent >= 0 and staging_percent <= 100),
	published_at timestamptz not null default now(),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	archived_at timestamptz,
	unique (app_id, version, channel)
);

create table if not exists artifacts (
	id uuid primary key,
	release_id uuid not null references releases(id) on delete cascade,
	platform varchar(40) not null,
	arch varchar(40) not null,
	file_type varchar(40) not null,
	file_name varchar(500) not null default '',
	file_url varchar(1000) not null,
	storage_key varchar(1000) not null,
	file_size bigint not null,
	sha512 varchar(256) not null,
	source_type varchar(40) not null default 'managed',
	anyshare_docid varchar(1000) not null default '',
	anyshare_rev varchar(255) not null default '',
	anyshare_name varchar(500) not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	archived_at timestamptz,
	unique (release_id, platform, arch, file_type)
);

create table if not exists admin_users (
	id uuid primary key,
	email varchar(255) not null unique,
	name varchar(160) not null default '',
	password_hash varchar(255) not null,
	is_active boolean not null default true,
	created_at timestamptz not null default now()
);

create table if not exists update_requests (
	id uuid primary key,
	app_id uuid references apps(id) on delete set null,
	app_slug varchar(80) not null,
	release_id uuid references releases(id) on delete set null,
	version varchar(80) not null default '',
	channel varchar(40) not null default '',
	platform varchar(40) not null default '',
	arch varchar(40) not null default '',
	client_id varchar(255) not null default '',
	format varchar(20) not null default 'json',
	matched boolean not null default false,
	staged_hit boolean not null default false,
	ip varchar(100) not null default '',
	user_agent text not null default '',
	created_at timestamptz not null default now()
);

create table if not exists download_events (
	id uuid primary key,
	app_id uuid not null references apps(id) on delete cascade,
	release_id uuid not null references releases(id) on delete cascade,
	artifact_id uuid not null references artifacts(id) on delete cascade,
	version varchar(80) not null,
	platform varchar(40) not null,
	arch varchar(40) not null,
	file_type varchar(40) not null,
	file_name varchar(500) not null default '',
	client_id varchar(255) not null default '',
	ip varchar(100) not null default '',
	user_agent text not null default '',
	created_at timestamptz not null default now()
);

alter table apps
	add column if not exists updated_at timestamptz not null default now(),
	add column if not exists archived_at timestamptz;

alter table releases
	add column if not exists updated_at timestamptz not null default now(),
	add column if not exists archived_at timestamptz;

alter table artifacts
	add column if not exists file_name varchar(500) not null default '',
	add column if not exists source_type varchar(40) not null default 'managed',
	add column if not exists anyshare_docid varchar(1000) not null default '',
	add column if not exists anyshare_rev varchar(255) not null default '',
	add column if not exists anyshare_name varchar(500) not null default '',
	add column if not exists updated_at timestamptz not null default now(),
	add column if not exists archived_at timestamptz;

alter table artifacts
	drop constraint if exists artifacts_release_id_platform_arch_file_type_key;

create index if not exists idx_releases_app_channel_published
	on releases (app_id, channel, published_at desc);
create index if not exists idx_artifacts_release
	on artifacts (release_id);
create index if not exists idx_admin_users_email
	on admin_users (lower(email));
create index if not exists idx_apps_active
	on apps (archived_at)
	where archived_at is null;
create index if not exists idx_releases_active_app_channel_published
	on releases (app_id, channel, published_at desc)
	where archived_at is null;
create index if not exists idx_artifacts_active_release
	on artifacts (release_id)
	where archived_at is null;
create index if not exists idx_update_requests_created
	on update_requests (created_at desc);
create index if not exists idx_update_requests_app_created
	on update_requests (app_id, created_at desc);
create index if not exists idx_download_events_created
	on download_events (created_at desc);
create index if not exists idx_download_events_app_created
	on download_events (app_id, created_at desc);
create index if not exists idx_download_events_release_created
	on download_events (release_id, created_at desc);
create index if not exists idx_artifacts_source_type
	on artifacts (source_type)
	where archived_at is null;
create unique index if not exists idx_artifacts_active_target_unique
	on artifacts (release_id, platform, arch, file_type)
	where archived_at is null;
`
