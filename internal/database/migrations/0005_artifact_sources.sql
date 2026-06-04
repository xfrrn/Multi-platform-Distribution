alter table artifacts
    add column if not exists source_type varchar(40) not null default 'managed',
    add column if not exists anyshare_docid varchar(1000) not null default '',
    add column if not exists anyshare_rev varchar(255) not null default '',
    add column if not exists anyshare_name varchar(500) not null default '';

create index if not exists idx_artifacts_source_type
    on artifacts (source_type)
    where archived_at is null;
