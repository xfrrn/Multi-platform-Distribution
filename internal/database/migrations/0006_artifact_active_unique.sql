alter table artifacts
    drop constraint if exists artifacts_release_id_platform_arch_file_type_key;

create unique index if not exists idx_artifacts_active_target_unique
    on artifacts (release_id, platform, arch, file_type)
    where archived_at is null;
