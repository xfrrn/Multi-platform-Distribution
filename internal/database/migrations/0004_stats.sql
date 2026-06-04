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

create index if not exists idx_update_requests_created
    on update_requests (created_at desc);

create index if not exists idx_update_requests_app_created
    on update_requests (app_id, created_at desc);

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

create index if not exists idx_download_events_created
    on download_events (created_at desc);

create index if not exists idx_download_events_app_created
    on download_events (app_id, created_at desc);

create index if not exists idx_download_events_release_created
    on download_events (release_id, created_at desc);
