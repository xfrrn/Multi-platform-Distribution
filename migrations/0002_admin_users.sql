create table if not exists admin_users (
    id uuid primary key,
    email varchar(255) not null unique,
    name varchar(160) not null default '',
    password_hash varchar(255) not null,
    is_active boolean not null default true,
    created_at timestamptz not null default now()
);

create index if not exists idx_admin_users_email
    on admin_users (lower(email));
