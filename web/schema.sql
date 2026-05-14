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
