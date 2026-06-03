create table if not exists apps (
    id uuid primary key,
    name varchar(160) not null,
    slug varchar(80) not null unique,
    description text not null default '',
    icon_url varchar(500) not null default '',
    default_channel varchar(40) not null default 'stable',
    created_at timestamptz not null default now()
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
    unique (app_id, version, channel)
);

create index if not exists idx_releases_app_channel_published
    on releases (app_id, channel, published_at desc);

create table if not exists artifacts (
    id uuid primary key,
    release_id uuid not null references releases(id) on delete cascade,
    platform varchar(40) not null,
    arch varchar(40) not null,
    file_type varchar(40) not null,
    file_url varchar(1000) not null,
    storage_key varchar(1000) not null,
    file_size bigint not null,
    sha512 varchar(256) not null,
    created_at timestamptz not null default now(),
    unique (release_id, platform, arch, file_type)
);

create index if not exists idx_artifacts_release
    on artifacts (release_id);
