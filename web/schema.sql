create table if not exists baseline_runs (
  id text primary key,
  workspace text not null,
  agent_kind text not null,
  status text not null,
  health_score integer not null,
  mode text not null,
  payload jsonb not null,
  created_at timestamptz not null default now()
);

create index if not exists baseline_runs_created_at_idx on baseline_runs (created_at desc);

create table if not exists baseline_events (
  id text primary key,
  type text not null,
  path text not null,
  payload jsonb not null,
  created_at timestamptz not null default now()
);

create index if not exists baseline_events_created_at_idx on baseline_events (created_at desc);

create table if not exists canonical_question_sets (
  slug text not null,
  version text not null,
  title text not null,
  questions jsonb not null,
  active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (slug, version)
);

create table if not exists llm_evaluations (
  id text primary key,
  run_id text not null,
  question_set_slug text not null,
  question_set_version text not null,
  model text not null,
  score integer not null,
  verdict text not null,
  payload jsonb not null,
  created_at timestamptz not null default now()
);

create index if not exists llm_evaluations_run_idx on llm_evaluations (run_id, created_at desc);

create table if not exists users (
  id text primary key,
  email text not null,
  email_normalized text not null unique,
  name text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists accounts (
  id text primary key,
  primary_user_id text not null references users(id),
  billing_email text not null,
  plan_key text not null default 'free',
  status text not null default 'pending',
  benchmark_consent boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists account_members (
  id text primary key,
  account_id text not null references accounts(id),
  user_id text not null references users(id),
  role text not null default 'owner',
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(account_id, user_id)
);

create table if not exists account_invites (
  id text primary key,
  account_id text not null references accounts(id),
  user_id text not null references users(id),
  email_normalized text not null,
  role text not null default 'owner',
  status text not null default 'pending',
  invited_by text,
  expires_at timestamptz not null,
  accepted_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists account_invites_email_idx on account_invites (email_normalized, created_at desc);

create table if not exists magic_links (
  id text primary key,
  user_id text not null references users(id),
  account_id text not null references accounts(id),
  email_normalized text not null,
  token_prefix text not null,
  token_hash text not null unique,
  purpose text not null,
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists magic_links_email_idx on magic_links (email_normalized, created_at desc);

create table if not exists account_sessions (
  id text primary key,
  user_id text not null references users(id),
  account_id text not null references accounts(id),
  token_prefix text not null,
  token_hash text not null unique,
  expires_at timestamptz not null,
  revoked_at timestamptz,
  last_seen_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists account_sessions_account_idx on account_sessions (account_id, created_at desc);

create table if not exists stripe_customers (
  id text primary key,
  account_id text not null references accounts(id),
  user_id text not null references users(id),
  stripe_customer_id text not null unique,
  email_normalized text not null,
  livemode boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists stripe_subscriptions (
  id text primary key,
  account_id text not null references accounts(id),
  stripe_customer_id text not null,
  stripe_subscription_id text not null unique,
  status text not null,
  price_id text not null,
  plan_key text not null,
  current_period_end timestamptz,
  cancel_at_period_end boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists entitlements (
  id text primary key,
  account_id text not null references accounts(id),
  key text not null,
  status text not null,
  source text not null,
  retention_days integer not null default 14,
  max_workspaces integer not null default 1,
  monitoring_enabled boolean not null default false,
  starts_at timestamptz,
  expires_at timestamptz,
  updated_at timestamptz not null default now(),
  unique(account_id, key)
);

create table if not exists workspaces (
  id text primary key,
  account_id text not null references accounts(id),
  workspace_hash text not null,
  display_name_redacted text not null,
  benchmark_consent boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(account_id, workspace_hash)
);

create index if not exists workspaces_account_idx on workspaces (account_id, updated_at desc);

create table if not exists api_tokens (
  id text primary key,
  account_id text not null references accounts(id),
  workspace_id text not null references workspaces(id),
  token_prefix text not null,
  token_hash text not null unique,
  scopes text[] not null,
  created_at timestamptz not null default now(),
  last_seen_at timestamptz,
  revoked_at timestamptz
);

create index if not exists api_tokens_workspace_idx on api_tokens (workspace_id, created_at desc);

create table if not exists stripe_events (
  id text primary key,
  stripe_event_id text not null unique,
  event_type text not null,
  livemode boolean not null default false,
  payload_hash text not null,
  processed_at timestamptz,
  status text not null default 'pending',
  error text,
  created_at timestamptz not null default now()
);

create table if not exists audit_log (
  id text primary key,
  actor_type text not null,
  actor_id text,
  action text not null,
  subject_type text not null,
  subject_id text,
  idempotency_key text,
  metadata_redacted jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists audit_log_subject_idx on audit_log (subject_type, subject_id, created_at desc);

create table if not exists lifecycle_event_outbox (
  id text primary key,
  provider text not null,
  event_name text not null,
  subject_type text not null,
  subject_id text not null,
  destination text not null,
  payload_redacted jsonb not null,
  idempotency_key text not null unique,
  status text not null default 'pending',
  attempts integer not null default 0,
  next_attempt_at timestamptz not null default now(),
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists comparison_aggregates (
  id text primary key,
  account_id text not null references accounts(id),
  workspace_id text not null references workspaces(id),
  check_id text not null,
  lane text,
  kind text,
  observations_count integer not null default 0,
  warning_count integer not null default 0,
  average_score numeric,
  p95_duration_ms integer,
  latest_run_id text,
  latest_status text,
  latest_seen_at timestamptz,
  aggregate_scope text not null default 'account-private',
  updated_at timestamptz not null default now(),
  unique(account_id, workspace_id, check_id)
);

alter table baseline_runs add column if not exists account_id text;
alter table baseline_runs add column if not exists workspace_id text;
alter table baseline_runs add column if not exists expires_at timestamptz;
alter table baseline_runs add column if not exists account_private_payload jsonb;
alter table baseline_runs add column if not exists comparison_scope text not null default 'legacy';

create index if not exists baseline_runs_account_created_idx on baseline_runs (account_id, created_at desc);
create index if not exists baseline_runs_workspace_created_idx on baseline_runs (workspace_id, created_at desc);
