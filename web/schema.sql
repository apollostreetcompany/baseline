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
