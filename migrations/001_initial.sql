CREATE TABLE IF NOT EXISTS clusters (
  cluster_id text PRIMARY KEY,
  creator_ip text NOT NULL DEFAULT '',
  creator_user_agent text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS instances (
  instance_id text PRIMARY KEY,
  cluster_id text NOT NULL REFERENCES clusters (cluster_id),
  flynn_version text NOT NULL DEFAULT '',
  ssh_public_keys jsonb NOT NULL DEFAULT '[]'::jsonb,
  url text NOT NULL,
  name text NOT NULL,
  creator_ip text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (cluster_id, url)
);
