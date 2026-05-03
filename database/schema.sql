create table if not exists users (
    user_ID bigint primary key,
    username text not null unique,
    password text not null,
    family_ID bigint not null
);

create table if not exists families (
    family_ID bigint primary key,
    owner_ID bigint not null,
    members jsonb not null default '[]'::jsonb,
    code text not null unique
);

create table if not exists tasks (
    task_ID bigint primary key,
    title text not null,
    completed boolean not null default false,
    completedBy text not null default '',
    user_ID bigint not null,
    family_ID bigint,
    scope text not null
);

create index if not exists idx_users_user_id on users (user_ID);
create index if not exists idx_users_family_id on users (family_ID);
create index if not exists idx_families_family_id on families (family_ID);
create index if not exists idx_tasks_user_id on tasks (user_ID);
create index if not exists idx_tasks_family_id on tasks (family_ID);
create index if not exists idx_tasks_scope on tasks (scope);