alter table apps
    add column if not exists updated_at timestamptz not null default now(),
    add column if not exists archived_at timestamptz;

alter table releases
    add column if not exists updated_at timestamptz not null default now(),
    add column if not exists archived_at timestamptz;

alter table artifacts
    add column if not exists file_name varchar(500) not null default '',
    add column if not exists updated_at timestamptz not null default now(),
    add column if not exists archived_at timestamptz;

create index if not exists idx_apps_active
    on apps (archived_at)
    where archived_at is null;

create index if not exists idx_releases_active_app_channel_published
    on releases (app_id, channel, published_at desc)
    where archived_at is null;

create index if not exists idx_artifacts_active_release
    on artifacts (release_id)
    where archived_at is null;
