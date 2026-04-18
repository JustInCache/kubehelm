import { api } from './client';
import type {
  AuditEvent, AuditStats, ChartInfo, Cluster, ClusterHealth, ComplianceCheck,
  HelmRelease, HelmRevision, HelmRepository, ProviderSpec,
  NotificationChannel, NotificationRule,
  Report, SettingsPayload, User
} from './types';

export const authApi = {
  async login(email: string, password: string) {
    const { data } = await api.post<{ token: string; refreshToken: string; user: User }>('/auth/login', { email, password });
    return data;
  },
  async me() {
    const { data } = await api.get<User>('/auth/me');
    return data;
  }
};

export const clustersApi = {
  async list() {
    const { data } = await api.get<Cluster[]>('/clusters');
    return data;
  },
  async get(clusterId: string) {
    const { data } = await api.get<Cluster>(`/clusters/${clusterId}`);
    return data;
  },
  async create(payload: { name: string; provider: string; environment: string; kubeconfig: string }) {
    const { data } = await api.post<Cluster>('/clusters', payload);
    return data;
  },
  async delete(clusterId: string) {
    await api.delete(`/clusters/${clusterId}`);
  },
  async health(clusterId: string) {
    const { data } = await api.get<ClusterHealth>(`/clusters/${clusterId}/health`);
    return data;
  },
  async nodes(clusterId: string) {
    const { data } = await api.get<Record<string, unknown>[]>(`/clusters/${clusterId}/nodes`);
    return data;
  },
  async namespaces(clusterId: string) {
    const { data } = await api.get<string[]>(`/clusters/${clusterId}/namespaces`);
    return data;
  },
  async testConnection(kubeconfig: string) {
    const { data } = await api.post<{ connected: boolean; serverVersion?: string; error?: string; checkedAt: string }>(
      '/clusters/test-connection', { kubeconfig });
    return data;
  }
};

export const releasesApi = {
  async list(params?: { namespace?: string; search?: string; page?: number; limit?: number; sortBy?: string; sortOrder?: string }) {
    const { data } = await api.get<HelmRelease[]>('/helm/releases', { params });
    return data;
  },
  async history(releaseId: string) {
    const { data } = await api.get<HelmRevision[]>(`/helm/releases/${encodeURIComponent(releaseId)}/history`);
    return data;
  },
  async manifest(releaseId: string, revision: number) {
    const { data } = await api.get<{ manifest: string }>(
      `/helm/releases/${encodeURIComponent(releaseId)}/history/${revision}/manifest`);
    return data;
  },
  async diff(releaseId: string, revA: number, revB: number) {
    const { data } = await api.get(`/helm/releases/${encodeURIComponent(releaseId)}/diff`, { params: { revA, revB } });
    return data;
  },
  async upgrade(releaseId: string, payload: { chart: string; version: string; values?: Record<string, string> }) {
    const { data } = await api.post<{ success: boolean; output: string; message: string }>(
      `/helm/releases/${encodeURIComponent(releaseId)}/upgrade`, payload);
    return data;
  },
  async rollback(releaseId: string, revision: number) {
    const { data } = await api.post<{ success: boolean; output: string; message: string }>(
      `/helm/releases/${encodeURIComponent(releaseId)}/rollback`, { revision });
    return data;
  },
  async test(releaseId: string) {
    const { data } = await api.post<{ success: boolean; output: string }>(
      `/helm/releases/${encodeURIComponent(releaseId)}/test`, {});
    return data;
  },
  async dryRun(releaseId: string, payload: { chart: string; version: string }) {
    const { data } = await api.post<{ manifest: string }>(
      `/helm/releases/${encodeURIComponent(releaseId)}/dry-run`, payload);
    return data;
  },
  async listApprovals(status?: string) {
    const { data } = await api.get<HelmRelease[]>('/helm/approvals', { params: status ? { status } : undefined });
    return data;
  },
  async values(releaseId: string) {
    const { data } = await api.get<{ values: string }>(`/helm/releases/${encodeURIComponent(releaseId)}/values`);
    return data;
  },
  async install(payload: { clusterId: string; namespace: string; releaseName: string; chart: string; version?: string; values?: Record<string, string> }) {
    const { data } = await api.post<{ success: boolean; output: string }>('/helm/releases', payload);
    return data;
  },
  async uninstall(releaseId: string) {
    const { data } = await api.delete<{ success: boolean; output: string }>(`/helm/releases/${encodeURIComponent(releaseId)}`);
    return data;
  },
  async listCharts(repoId: string) {
    const { data } = await api.get<ChartInfo[]>('/helm/charts', { params: { repoId } });
    return data;
  }
};

export const auditApi = {
  async listEvents(params?: { page?: number; limit?: number }) {
    const { data } = await api.get<AuditEvent[]>('/audit/events', { params });
    return data;
  },
  async stats(period = '24 hours') {
    const { data } = await api.get<AuditStats>('/audit/stats', { params: { period } });
    return data;
  },
  async compliance() {
    const { data } = await api.get<ComplianceCheck[]>('/audit/compliance');
    return data;
  }
};

export const notificationsApi = {
  async listChannels() {
    const { data } = await api.get<NotificationChannel[]>('/notifications/channels');
    return data;
  },
  async createChannel(payload: { name: string; type: string }) {
    const { data } = await api.post<NotificationChannel>('/notifications/channels', payload);
    return data;
  },
  async toggleChannel(channelId: string, enabled: boolean) {
    const { data } = await api.put<NotificationChannel>(`/notifications/channels/${channelId}`, { enabled });
    return data;
  },
  async deleteChannel(channelId: string) {
    await api.delete(`/notifications/channels/${channelId}`);
  },
  async listRules() {
    const { data } = await api.get<NotificationRule[]>('/notifications/rules');
    return data;
  },
  async createRule(payload: { name: string; events: string[]; channelIds: string[]; filters?: Record<string, unknown> }) {
    const { data } = await api.post<NotificationRule>('/notifications/rules', payload);
    return data;
  }
};

export const reportsApi = {
  async list() {
    const { data } = await api.get<Report[]>('/reports');
    return data;
  },
  async create(payload: { name: string; type: string; format: string; filters?: Record<string, unknown> }) {
    const { data } = await api.post<Report>('/reports', payload);
    return data;
  }
};

export const providersApi = {
  async list() {
    const { data } = await api.get<ProviderSpec[]>('/helm/providers');
    return data;
  },
  async listEnabled() {
    const { data } = await api.get<string[]>('/helm/providers/enabled');
    return data;
  },
  async install(providerId: string) {
    const { data } = await api.post<{ message: string; enabledProviders: string[] }>('/helm/providers/install', { providerId });
    return data;
  },
  async uninstall(providerId: string) {
    const { data } = await api.post<{ message: string; enabledProviders: string[] }>('/helm/providers/uninstall', { providerId });
    return data;
  }
};

export const repositoriesApi = {
  async list() {
    const { data } = await api.get<HelmRepository[]>('/helm/repositories');
    return data;
  },
  async add(payload: { name: string; url: string; providerId: string; credentials: Record<string, string> }) {
    const { data } = await api.post<HelmRepository>('/helm/repositories', payload);
    return data;
  },
  async update(repoId: string, payload: { url: string; credentials: Record<string, string> }) {
    const { data } = await api.put<HelmRepository>(`/helm/repositories/${repoId}`, payload);
    return data;
  },
  async remove(repoId: string) {
    await api.delete(`/helm/repositories/${repoId}`);
  },
  async refresh(repoId: string) {
    const { data } = await api.post<{ message: string }>(`/helm/repositories/${repoId}/refresh`);
    return data;
  },
  async test(repoId: string) {
    const { data } = await api.post<{ ok: boolean; error?: string }>(`/helm/repositories/${repoId}/test`);
    return data;
  }
};

export const settingsApi = {
  async get() {
    const { data } = await api.get<SettingsPayload>('/settings');
    return data;
  },
  async updateOrganization(payload: { name?: string; settings?: Record<string, unknown> }) {
    const { data } = await api.put('/settings/organization', payload);
    return data;
  },
  async updateUserRole(userId: string, role: string) {
    const { data } = await api.put<User>(`/settings/users/${userId}/role`, { role });
    return data;
  }
};
