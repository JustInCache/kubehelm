export type User = {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'sre' | 'readonly';
  orgId: string;
};

export type Cluster = {
  id: string;
  orgId: string;
  name: string;
  provider: string;
  environment: string;
  authType: string;
  status: string;
  serverVersion?: string;
  lastError?: string;
  releaseCount?: number;
  nodeCount?: number;
  createdAt: string;
};

export type ClusterHealth = {
  nodes: { total: number; ready: number; notReady: number };
  pods: { total: number; running: number; pending: number; failed: number; succeeded: number };
  timestamp: string;
  error?: string;
};

export type HelmRelease = {
  id: string;
  clusterId: string;
  clusterName: string;
  name: string;
  namespace: string;
  chartName: string;
  chartVersion: string;
  appVersion?: string;
  status: string;
  revision: number;
  updatedAt?: string;
  createdAt: string;
};

export type ChartInfo = {
  name: string;
  version: string;
  appVersion?: string;
  description?: string;
  keywords?: string[];
  icon?: string;
  repoId?: string;
  repoName?: string;
};

export type HelmRevision = {
  id: string;
  releaseId: string;
  revision: number;
  chartVersion: string;
  status: string;
  description?: string;
  manifest?: string;
  deployedAt: string;
};

export type AuditEvent = {
  id: string;
  orgId?: string;
  clusterId?: string;
  clusterName?: string;
  username: string;
  action: string;
  resourceType: string;
  resourceName?: string;
  namespace?: string;
  sourceIp?: string;
  createdAt: string;
};

export type AuditStats = {
  actionStats: Array<{ action: string; count: number }>;
  resourceStats: Array<{ resource_type: string; count: number }>;
  userStats: Array<{ username: string; count: number }>;
  timeline: Array<{ hour: string; count: number }>;
};

export type ComplianceCheck = {
  id: string;
  category: string;
  name: string;
  status: string;
  message?: string;
  checkedAt: string;
};

export type NotificationChannel = {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  createdAt: string;
};

export type NotificationRule = {
  id: string;
  name: string;
  events: string[];
  channelIds: string[];
  enabled: boolean;
  createdAt: string;
};

export type Report = {
  id: string;
  name: string;
  type: string;
  status: string;
  format: string;
  fileSize?: number;
  createdAt: string;
};

export type SettingsPayload = {
  organization: {
    id: string;
    name: string;
    settings?: Record<string, unknown>;
  };
  users: User[];
};

export type FieldSpec = {
  key: string;
  label: string;
  placeholder: string;
  required: boolean;
  secret: boolean;
};

export type ProviderSpec = {
  id: string;
  name: string;
  description: string;
  icon: string;
  category: string;
  isOci: boolean;
  fields: FieldSpec[];
};

export type HelmRepository = {
  id: string;
  orgId: string;
  name: string;
  url: string;
  providerId: string;
  providerName?: string;
  status: string;
  lastError?: string;
  lastSync?: string;
  createdAt: string;
};
