CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  settings JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'readonly',
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS clusters (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  provider VARCHAR(50) NOT NULL,
  environment VARCHAR(50) NOT NULL,
  auth_type VARCHAR(50) NOT NULL DEFAULT 'kubeconfig',
  status VARCHAR(50) NOT NULL DEFAULT 'pending',
  server_version VARCHAR(100),
  last_error TEXT,
  kubeconfig_raw TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS helm_releases (
  id UUID PRIMARY KEY,
  cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  namespace VARCHAR(255) NOT NULL,
  chart_name VARCHAR(255),
  chart_version VARCHAR(255),
  status VARCHAR(100),
  revision INTEGER DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS helm_release_history (
  id UUID PRIMARY KEY,
  release_id UUID NOT NULL REFERENCES helm_releases(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL,
  chart_version VARCHAR(255),
  status VARCHAR(100),
  description TEXT,
  manifest TEXT,
  values_yaml TEXT,
  deployed_at TIMESTAMPTZ,
  UNIQUE(release_id, revision)
);

CREATE TABLE IF NOT EXISTS release_approvals (
  id UUID PRIMARY KEY,
  release_id UUID NOT NULL REFERENCES helm_releases(id) ON DELETE CASCADE,
  requested_by UUID NOT NULL REFERENCES users(id),
  reviewed_by UUID REFERENCES users(id),
  target_version VARCHAR(255),
  status VARCHAR(50) NOT NULL DEFAULT 'pending',
  rejection_reason TEXT,
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS drift_detections (
  id UUID PRIMARY KEY,
  release_id UUID NOT NULL REFERENCES helm_releases(id) ON DELETE CASCADE,
  status VARCHAR(50) NOT NULL DEFAULT 'ok',
  diff TEXT,
  detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_events (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  cluster_id UUID REFERENCES clusters(id) ON DELETE SET NULL,
  username VARCHAR(255),
  action VARCHAR(100) NOT NULL,
  resource_type VARCHAR(255) NOT NULL,
  resource_name VARCHAR(255),
  namespace VARCHAR(255),
  details JSONB DEFAULT '{}',
  source_ip INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance_checks (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  cluster_id UUID REFERENCES clusters(id) ON DELETE CASCADE,
  category VARCHAR(100) NOT NULL,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) NOT NULL DEFAULT 'unknown',
  message TEXT,
  details JSONB DEFAULT '{}',
  checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_channels (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  type VARCHAR(50) NOT NULL,
  config_enc TEXT NOT NULL DEFAULT '{}',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_rules (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  events JSONB NOT NULL DEFAULT '[]',
  channel_ids JSONB NOT NULL DEFAULT '[]',
  filters JSONB DEFAULT '{}',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  type VARCHAR(100) NOT NULL,
  format VARCHAR(10) NOT NULL,
  status VARCHAR(50) NOT NULL DEFAULT 'pending',
  filters JSONB DEFAULT '{}',
  file_size INTEGER,
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

INSERT INTO organizations (id, name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Default Organization')
ON CONFLICT DO NOTHING;

INSERT INTO users (id, org_id, email, password_hash, name, role, is_active) VALUES
  ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'admin@kubeaudit.io',
   '$2a$12$BbnwXOn26Gw2RaCfJ8VHtuQR2IqS8YY8dJIpfY3tGo3kkMXw3Qs8m', 'Admin User', 'admin', true)
ON CONFLICT DO NOTHING;

-- No mock cluster/release seed data — clusters are registered at runtime
ON CONFLICT DO NOTHING;

INSERT INTO notification_channels (id, org_id, name, type, config_enc, enabled, created_at) VALUES
  ('66666666-6666-6666-6666-666666666666', '00000000-0000-0000-0000-000000000001', 'Ops Slack', 'slack', '{}', true, NOW() - INTERVAL '1 day')
ON CONFLICT DO NOTHING;

INSERT INTO notification_rules (id, org_id, name, events, channel_ids, filters, enabled, created_at) VALUES
  ('77777777-7777-7777-7777-777777777777', '00000000-0000-0000-0000-000000000001', 'Drift Alerts', '["drift.detected"]', '["66666666-6666-6666-6666-666666666666"]', '{"environment":"prod"}', true, NOW() - INTERVAL '1 day')
ON CONFLICT DO NOTHING;

INSERT INTO reports (id, org_id, name, type, format, status, file_size, created_by, created_at, completed_at) VALUES
  ('88888888-8888-8888-8888-888888888888', '00000000-0000-0000-0000-000000000001', 'Daily Audit', 'audit', 'csv', 'completed', 2048, '00000000-0000-0000-0000-000000000002', NOW() - INTERVAL '8 hour', NOW() - INTERVAL '7 hour')
ON CONFLICT DO NOTHING;


-- Helm repository management
CREATE TABLE IF NOT EXISTS helm_repositories (
  id               UUID PRIMARY KEY,
  org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name             VARCHAR(255) NOT NULL,
  url              TEXT NOT NULL,
  provider_id      VARCHAR(100) NOT NULL,
  credentials_json TEXT,
  status           VARCHAR(50) NOT NULL DEFAULT 'pending',
  last_error       TEXT,
  last_sync        TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(org_id, name)
);
