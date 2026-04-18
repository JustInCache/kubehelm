import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { clustersApi } from '../lib/api/endpoints';
import { statusBadgeClass, formatDate } from '../lib/utils';
import type { Cluster } from '../lib/api/types';

function AddClusterModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [name, setName] = useState('');
  const [provider, setProvider] = useState('eks');
  const [environment, setEnvironment] = useState('prod');
  const [kubeconfig, setKubeconfig] = useState('');
  const [testResult, setTestResult] = useState<{ connected: boolean; serverVersion?: string; error?: string } | null>(null);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState('');

  const createMut = useMutation({
    mutationFn: () => clustersApi.create({ name, provider, environment, kubeconfig }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['clusters'] }); onClose(); },
    onError: (e: unknown) => setError((e as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to create cluster'),
  });

  const handleTest = async () => {
    if (!kubeconfig.trim()) { setError('Paste a kubeconfig first'); return; }
    setTesting(true); setError(''); setTestResult(null);
    try {
      const res = await clustersApi.testConnection(kubeconfig);
      setTestResult(res);
    } catch {
      setTestResult({ connected: false, error: 'Connection failed' });
    } finally { setTesting(false); }
  };

  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div className="modal-header">
          <span className="modal-title">Connect Kubernetes Cluster</span>
          <button className="btn btn-ghost btn-icon" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          {error && <div className="alert alert-error">{error}</div>}
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Cluster Name *</label>
              <input className="form-input" value={name} onChange={e => setName(e.target.value)} placeholder="prod-eks-us-east-1" />
            </div>
            <div className="form-group">
              <label className="form-label">Provider</label>
              <select className="form-select" value={provider} onChange={e => setProvider(e.target.value)}>
                <option value="eks">Amazon EKS</option>
                <option value="gke">Google GKE</option>
                <option value="aks">Azure AKS</option>
                <option value="k3s">k3s / k3d</option>
                <option value="kind">kind</option>
                <option value="other">Other</option>
              </select>
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Environment</label>
            <select className="form-select" value={environment} onChange={e => setEnvironment(e.target.value)}>
              <option value="prod">Production</option>
              <option value="staging">Staging</option>
              <option value="dev">Development</option>
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Kubeconfig (YAML)</label>
            <textarea
              className="form-textarea"
              value={kubeconfig}
              onChange={e => setKubeconfig(e.target.value)}
              placeholder="Paste your kubeconfig YAML here..."
              rows={8}
            />
            <div className="form-hint">The kubeconfig is stored securely and used to communicate with the cluster API.</div>
          </div>
          {testResult && (
            <div className={`alert ${testResult.connected ? 'alert-success' : 'alert-error'}`}>
              {testResult.connected
                ? `✓ Connected — Server version: ${testResult.serverVersion}`
                : `✗ Connection failed: ${testResult.error}`}
            </div>
          )}
          <button className="btn btn-ghost" onClick={handleTest} disabled={testing || !kubeconfig.trim()}>
            {testing ? '⟳ Testing…' : '⚡ Test Connection'}
          </button>
        </div>
        <div className="modal-footer">
          <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button
            className="btn btn-primary"
            onClick={() => createMut.mutate()}
            disabled={!name.trim() || createMut.isPending}
          >
            {createMut.isPending ? '⟳ Connecting…' : 'Add Cluster'}
          </button>
        </div>
      </div>
    </div>
  );
}

function ClusterRow({ cluster, onDelete }: { cluster: Cluster; onDelete: (id: string) => void }) {
  const [expanded, setExpanded] = useState(false);
  const health = useQuery({
    queryKey: ['cluster-health', cluster.id],
    queryFn: () => clustersApi.health(cluster.id),
    enabled: expanded && cluster.status === 'connected',
  });

  return (
    <>
      <tr onClick={() => setExpanded(e => !e)} style={{ cursor: 'pointer' }}>
        <td>
          <div className="col-name">{cluster.name}</div>
          <div className="col-muted">{cluster.serverVersion || '—'}</div>
        </td>
        <td><span className="badge badge-blue">{cluster.provider}</span></td>
        <td><span className="badge badge-purple">{cluster.environment}</span></td>
        <td><span className={`badge ${statusBadgeClass(cluster.status)}`}>{cluster.status}</span></td>
        <td className="text-muted">{cluster.releaseCount ?? 0}</td>
        <td className="col-muted">{formatDate(cluster.createdAt)}</td>
        <td>
          <button
            className="btn btn-danger btn-sm"
            onClick={e => { e.stopPropagation(); onDelete(cluster.id); }}
          >
            Delete
          </button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={7} style={{ background: 'var(--bg3)', padding: '12px 16px' }}>
            {health.isLoading && <div className="text-muted text-sm">Loading health data…</div>}
            {health.data && !health.data.error && (
              <div className="flex gap-3">
                <div className="card" style={{ flex: 1, padding: '10px 14px' }}>
                  <div className="card-title">Nodes</div>
                  <div style={{ fontSize: 24, fontWeight: 700 }}>{health.data.nodes.total}</div>
                  <div className="text-sm text-muted">
                    <span className="text-success">{health.data.nodes.ready} ready</span>
                    {health.data.nodes.notReady > 0 && <span className="text-danger"> · {health.data.nodes.notReady} not ready</span>}
                  </div>
                </div>
                <div className="card" style={{ flex: 1, padding: '10px 14px' }}>
                  <div className="card-title">Pods</div>
                  <div style={{ fontSize: 24, fontWeight: 700 }}>{health.data.pods.total}</div>
                  <div className="text-sm text-muted">
                    <span className="text-success">{health.data.pods.running} running</span>
                    {health.data.pods.pending > 0 && <span className="text-warning"> · {health.data.pods.pending} pending</span>}
                    {health.data.pods.failed > 0 && <span className="text-danger"> · {health.data.pods.failed} failed</span>}
                  </div>
                </div>
              </div>
            )}
            {health.data?.error && <div className="alert alert-error">{health.data.error}</div>}
            {cluster.lastError && <div className="alert alert-error mt-2">{cluster.lastError}</div>}
          </td>
        </tr>
      )}
    </>
  );
}

export default function ClustersPage() {
  const qc = useQueryClient();
  const [showAdd, setShowAdd] = useState(false);
  const { data: clusters, isLoading, isError } = useQuery({
    queryKey: ['clusters'],
    queryFn: () => clustersApi.list(),
    refetchInterval: 30000,
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => clustersApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['clusters'] }),
  });

  const handleDelete = (id: string) => {
    if (window.confirm('Delete this cluster? Releases will no longer be managed.')) {
      deleteMut.mutate(id);
    }
  };

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Clusters</div>
          <div className="page-subtitle">Manage your Kubernetes cluster connections</div>
        </div>
        <button className="btn btn-primary" onClick={() => setShowAdd(true)}>
          + Add Cluster
        </button>
      </div>

      {showAdd && <AddClusterModal onClose={() => setShowAdd(false)} />}

      {isLoading && <div className="loading">Loading clusters…</div>}
      {isError && <div className="alert alert-error">Failed to load clusters</div>}

      {!isLoading && !clusters?.length && (
        <div className="empty-state">
          <div className="empty-state-icon">⊞</div>
          <h3>No clusters yet</h3>
          <p>Connect your first Kubernetes cluster to start managing Helm releases.</p>
          <button className="btn btn-primary" onClick={() => setShowAdd(true)}>Add Cluster</button>
        </div>
      )}

      {clusters && clusters.length > 0 && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Cluster</th><th>Provider</th><th>Environment</th>
                  <th>Status</th><th>Releases</th><th>Added</th><th></th>
                </tr>
              </thead>
              <tbody>
                {clusters.map(c => (
                  <ClusterRow key={c.id} cluster={c} onDelete={handleDelete} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
