import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { providersApi, repositoriesApi } from '../lib/api/endpoints';
import { statusBadgeClass, formatDistanceToNow } from '../lib/utils';
import type { ProviderSpec, HelmRepository } from '../lib/api/types';

// ── Add Repository Modal ─────────────────────────────────────────────────────

function AddRepoModal({
  providers,
  enabledIds,
  onClose,
}: {
  providers: ProviderSpec[];
  enabledIds: string[];
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [step, setStep] = useState<'pick' | 'form'>('pick');
  const [selectedProvider, setSelectedProvider] = useState<ProviderSpec | null>(null);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [creds, setCreds] = useState<Record<string, string>>({});
  const [error, setError] = useState('');

  const installed = providers.filter(p => enabledIds.includes(p.id));

  const addMut = useMutation({
    mutationFn: () =>
      repositoriesApi.add({
        name,
        url: selectedProvider?.isOci ? '' : url,
        providerId: selectedProvider!.id,
        credentials: creds,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['helm-repos'] });
      onClose();
    },
    onError: (e: unknown) =>
      setError((e as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to add repository'),
  });

  const urlHints: Record<string, string> = {
    harbor:      'Harbor: use the chartrepo project path — e.g. http://harbor.host/chartrepo/library',
    artifactory: 'Artifactory: include the repo key — e.g. http://art.host/artifactory/helm-local',
    nexus:       'Nexus: use the repository path — e.g. http://nexus.host/repository/helm-hosted',
  };

  const urlField = selectedProvider && !selectedProvider.isOci ? (
    <div className="form-group">
      <label className="form-label">Repository URL <span style={{ color: 'var(--danger)' }}>*</span></label>
      <input
        className="form-input"
        value={url}
        onChange={e => setUrl(e.target.value)}
        placeholder={selectedProvider.fields.find(f => f.key === 'url')?.placeholder ?? 'https://…'}
      />
      {urlHints[selectedProvider.id] && (
        <div className="form-hint" style={{ color: 'var(--warning)', marginTop: 4 }}>
          ⚠ {urlHints[selectedProvider.id]}
        </div>
      )}
    </div>
  ) : null;

  return (
    <div className="modal-overlay" onClick={e => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: 540 }}>
        <div className="modal-header">
          <span className="modal-title">
            {step === 'pick' ? 'Select Provider' : `Add ${selectedProvider?.name} Repository`}
          </span>
          <button className="btn btn-ghost btn-icon" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          {step === 'pick' && (
            <>
              {installed.length === 0 && (
                <div className="alert alert-warning">
                  No providers installed. Go to the Plugins tab and install at least one provider first.
                </div>
              )}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                {installed.map(p => (
                  <button
                    key={p.id}
                    className="provider-pick-card"
                    onClick={() => { setSelectedProvider(p); setStep('form'); }}
                  >
                    <span className="provider-icon">{p.icon}</span>
                    <span className="provider-pick-name">{p.name}</span>
                    <span className="provider-pick-cat">{p.category}</span>
                  </button>
                ))}
              </div>
            </>
          )}

          {step === 'form' && selectedProvider && (
            <>
              {error && <div className="alert alert-error">{error}</div>}
              <div className="form-group">
                <label className="form-label">Repository Name <span style={{ color: 'var(--danger)' }}>*</span></label>
                <input
                  className="form-input"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder={`my-${selectedProvider.id}-repo`}
                />
              </div>
              {urlField}
              {selectedProvider.fields
                .filter(f => f.key !== 'url')
                .map(f => (
                  <div className="form-group" key={f.key}>
                    <label className="form-label">
                      {f.label}
                      {f.required && <span style={{ color: 'var(--danger)' }}> *</span>}
                    </label>
                    <input
                      className="form-input"
                      type={f.secret ? 'password' : 'text'}
                      placeholder={f.placeholder}
                      value={creds[f.key] ?? ''}
                      onChange={e => setCreds(prev => ({ ...prev, [f.key]: e.target.value }))}
                    />
                  </div>
                ))}
              {selectedProvider.isOci && (
                <div className="alert alert-info" style={{ fontSize: 12 }}>
                  ECR uses OCI — the registry URL is derived automatically from Account ID + Region.
                </div>
              )}
            </>
          )}
        </div>
        <div className="modal-footer">
          {step === 'form' && (
            <button className="btn btn-ghost" onClick={() => { setStep('pick'); setError(''); }}>
              ← Back
            </button>
          )}
          <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
          {step === 'form' && (
            <button
              className="btn btn-primary"
              disabled={!name || addMut.isPending}
              onClick={() => addMut.mutate()}
            >
              {addMut.isPending ? '⟳ Adding…' : 'Add Repository'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Edit Repository Modal ─────────────────────────────────────────────────────

function EditRepoModal({
  repo,
  provider,
  onClose,
}: {
  repo: HelmRepository;
  provider: ProviderSpec | undefined;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [url, setUrl] = useState(repo.url || '');
  const [creds, setCreds] = useState<Record<string, string>>({});
  const [error, setError] = useState('');

  const updateMut = useMutation({
    mutationFn: () => repositoriesApi.update(repo.id, { url, credentials: creds }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['helm-repos'] });
      onClose();
    },
    onError: (e: unknown) =>
      setError((e as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to update repository'),
  });

  const urlHints: Record<string, string> = {
    harbor:      'Harbor: use the chartrepo project path — e.g. http://harbor.host/chartrepo/library',
    artifactory: 'Artifactory: include the repo key — e.g. http://art.host/artifactory/helm-local',
    nexus:       'Nexus: use the repository path — e.g. http://nexus.host/repository/helm-hosted',
  };

  return (
    <div className="modal-overlay" onClick={e => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: 540 }}>
        <div className="modal-header">
          <span className="modal-title">Edit Repository — {repo.name}</span>
          <button className="btn btn-ghost btn-icon" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          {error && <div className="alert alert-error" style={{ marginBottom: 12 }}>{error}</div>}

          <div className="form-group">
            <label className="form-label">Provider</label>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: 'var(--bg2)', borderRadius: 6, fontSize: 13, color: 'var(--text2)' }}>
              <span>{provider?.icon}</span>
              <span>{provider?.name || repo.providerId}</span>
              <span className="badge badge-blue" style={{ fontSize: 10, marginLeft: 'auto' }}>{provider?.isOci ? 'OCI' : 'HTTP'}</span>
            </div>
          </div>

          {provider && !provider.isOci && (
            <div className="form-group">
              <label className="form-label">Repository URL <span style={{ color: 'var(--danger)' }}>*</span></label>
              <input
                className="form-input"
                value={url}
                onChange={e => setUrl(e.target.value)}
                placeholder={provider.fields.find(f => f.key === 'url')?.placeholder ?? 'https://…'}
              />
              {urlHints[provider.id] && (
                <div className="form-hint" style={{ color: 'var(--warning)', marginTop: 4 }}>
                  ⚠ {urlHints[provider.id]}
                </div>
              )}
            </div>
          )}

          {provider?.fields.filter(f => f.key !== 'url').map(f => (
            <div className="form-group" key={f.key}>
              <label className="form-label">{f.label}</label>
              <input
                className="form-input"
                type={f.secret ? 'password' : 'text'}
                placeholder={f.secret ? 'Leave blank to keep existing value' : f.placeholder}
                value={creds[f.key] ?? ''}
                onChange={e => setCreds(prev => ({ ...prev, [f.key]: e.target.value }))}
              />
            </div>
          ))}

          {provider?.isOci && (
            <div className="alert alert-info" style={{ fontSize: 12 }}>
              ECR uses OCI — update credentials below. The registry URL is derived automatically.
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button
            className="btn btn-primary"
            disabled={updateMut.isPending}
            onClick={() => updateMut.mutate()}
          >
            {updateMut.isPending ? '⟳ Saving…' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Plugins Tab ──────────────────────────────────────────────────────────────

function pluginError(e: unknown): string {
  const err = e as { uiMessage?: string; response?: { data?: { error?: string } } };
  return err?.uiMessage ?? err?.response?.data?.error ?? 'Operation failed';
}

function PluginsTab() {
  const qc = useQueryClient();
  const [mutError, setMutError] = useState('');

  const { data: allProviders = [], isLoading } = useQuery({
    queryKey: ['providers'],
    queryFn: providersApi.list,
  });

  const { data: enabledIds = [] } = useQuery({
    queryKey: ['providers-enabled'],
    queryFn: providersApi.listEnabled,
  });

  const installMut = useMutation({
    mutationFn: (id: string) => providersApi.install(id),
    onSuccess: () => { setMutError(''); qc.invalidateQueries({ queryKey: ['providers-enabled'] }); },
    onError: (e: unknown) => setMutError(pluginError(e)),
  });

  const uninstallMut = useMutation({
    mutationFn: (id: string) => providersApi.uninstall(id),
    onSuccess: () => { setMutError(''); qc.invalidateQueries({ queryKey: ['providers-enabled'] }); },
    onError: (e: unknown) => setMutError(pluginError(e)),
  });

  if (isLoading) return <div className="text-muted text-sm">Loading providers…</div>;

  const selfHosted = allProviders.filter(p => p.category === 'self-hosted');
  const cloud = allProviders.filter(p => p.category === 'cloud');

  const isBusy = installMut.isPending || uninstallMut.isPending;

  const renderGroup = (label: string, list: ProviderSpec[]) => (
    <div style={{ marginBottom: 24 }}>
      <div className="fw-600 text-sm" style={{ marginBottom: 10, color: 'var(--text2)', textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: 11 }}>
        {label}
      </div>
      <div className="provider-grid">
        {list.map(p => {
          const isEnabled = enabledIds.includes(p.id);
          return (
            <div key={p.id} className={`provider-card${isEnabled ? ' enabled' : ''}`}>
              <div className="provider-card-header">
                <span className="provider-icon">{p.icon}</span>
                <div style={{ flex: 1 }}>
                  <div className="fw-600" style={{ fontSize: 14 }}>{p.name}</div>
                  <span className={`badge ${p.isOci ? 'badge-purple' : 'badge-blue'}`} style={{ fontSize: 10 }}>
                    {p.isOci ? 'OCI' : 'HTTP'}
                  </span>
                </div>
                {isEnabled && <span className="badge badge-green" style={{ fontSize: 10 }}>Installed</span>}
              </div>
              <p className="provider-desc">{p.description}</p>
              <div style={{ display: 'flex', gap: 8, marginTop: 'auto' }}>
                {isEnabled ? (
                  <button
                    className="btn btn-danger btn-sm"
                    style={{ flex: 1 }}
                    disabled={isBusy}
                    onClick={() => uninstallMut.mutate(p.id)}
                  >
                    {uninstallMut.isPending && uninstallMut.variables === p.id ? 'Removing…' : 'Uninstall'}
                  </button>
                ) : (
                  <button
                    className="btn btn-primary btn-sm"
                    style={{ flex: 1 }}
                    disabled={isBusy}
                    onClick={() => installMut.mutate(p.id)}
                  >
                    {installMut.isPending && installMut.variables === p.id ? 'Installing…' : 'Install'}
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );

  return (
    <div>
      {mutError && (
        <div className="alert alert-error" style={{ marginBottom: 16, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span>{mutError}</span>
          <button onClick={() => setMutError('')} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 16, color: 'inherit' }}>✕</button>
        </div>
      )}
      {renderGroup('Self-Hosted', selfHosted)}
      {renderGroup('Cloud', cloud)}
    </div>
  );
}

// ── Repositories Tab ─────────────────────────────────────────────────────────

function RepositoriesTab({ enabledIds, allProviders }: { enabledIds: string[]; allProviders: ProviderSpec[] }) {
  const qc = useQueryClient();
  const [showAdd, setShowAdd] = useState(false);
  const [editRepo, setEditRepo] = useState<HelmRepository | null>(null);

  const { data: repos = [], isLoading, isError } = useQuery({
    queryKey: ['helm-repos'],
    queryFn: repositoriesApi.list,
    refetchInterval: 30000,
  });

  const removeMut = useMutation({
    mutationFn: (id: string) => repositoriesApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['helm-repos'] }),
  });

  const refreshMut = useMutation({
    mutationFn: (id: string) => repositoriesApi.refresh(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['helm-repos'] }),
  });

  const testMut = useMutation({
    mutationFn: (id: string) => repositoriesApi.test(id),
    onSuccess: (data, id) => {
      alert(data.ok ? `Repository ${id}: connection OK` : `Connection failed: ${data.error}`);
    },
  });

  return (
    <div>
      {showAdd && (
        <AddRepoModal
          providers={allProviders}
          enabledIds={enabledIds}
          onClose={() => setShowAdd(false)}
        />
      )}
      {editRepo && (
        <EditRepoModal
          repo={editRepo}
          provider={allProviders.find(p => p.id === editRepo.providerId)}
          onClose={() => setEditRepo(null)}
        />
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <button className="btn btn-primary btn-sm" onClick={() => setShowAdd(true)}>
          + Add Repository
        </button>
      </div>

      {isError && <div className="alert alert-error">Failed to load repositories</div>}

      {!isLoading && repos.length === 0 && (
        <div className="empty-state">
          <div className="empty-state-icon">⊗</div>
          <h3>No repositories configured</h3>
          <p>Install a provider plugin and add a repository to browse and deploy Helm charts.</p>
        </div>
      )}

      {repos.length > 0 && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Provider</th>
                  <th>URL / Registry</th>
                  <th>Status</th>
                  <th>Last Sync</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {repos.map((r: HelmRepository) => (
                  <tr key={r.id}>
                    <td className="col-name">{r.name}</td>
                    <td>
                      <span className="badge badge-blue">{r.providerName || r.providerId}</span>
                    </td>
                    <td className="col-mono" style={{ maxWidth: 240, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {r.url || '(OCI registry)'}
                    </td>
                    <td>
                      <span className={`badge ${statusBadgeClass(r.status)}`}>{r.status}</span>
                      {r.lastError && (
                        <div style={{ marginTop: 4, fontSize: 11, color: 'var(--danger)', maxWidth: 260, whiteSpace: 'normal', lineHeight: 1.3 }}>
                          {r.lastError}
                        </div>
                      )}
                    </td>
                    <td className="col-muted">
                      {r.lastSync ? formatDistanceToNow(r.lastSync) : '—'}
                    </td>
                    <td>
                      <div className="flex gap-2">
                        <button
                          className="btn btn-ghost btn-sm"
                          onClick={() => setEditRepo(r)}
                          title="Edit URL and credentials"
                        >
                          ✏
                        </button>
                        <button
                          className="btn btn-ghost btn-sm"
                          disabled={refreshMut.isPending}
                          onClick={() => refreshMut.mutate(r.id)}
                          title="Refresh index"
                        >
                          ↻
                        </button>
                        <button
                          className="btn btn-ghost btn-sm"
                          disabled={testMut.isPending}
                          onClick={() => testMut.mutate(r.id)}
                          title="Test connection"
                        >
                          Test
                        </button>
                        <button
                          className="btn btn-danger btn-sm"
                          disabled={removeMut.isPending}
                          onClick={() => {
                            if (confirm(`Remove repository "${r.name}"?`)) removeMut.mutate(r.id);
                          }}
                        >
                          Remove
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function RepositoriesPage() {
  const [tab, setTab] = useState<'plugins' | 'repos'>('plugins');

  const { data: allProviders = [] } = useQuery({
    queryKey: ['providers'],
    queryFn: providersApi.list,
  });

  const { data: enabledIds = [] } = useQuery({
    queryKey: ['providers-enabled'],
    queryFn: providersApi.listEnabled,
  });

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Repositories</div>
          <div className="page-subtitle">Manage Helm chart repository provider plugins and configured repositories</div>
        </div>
      </div>

      <div className="tab-bar">
        <button
          className={`tab-btn${tab === 'plugins' ? ' active' : ''}`}
          onClick={() => setTab('plugins')}
        >
          Plugins
          <span className="tab-badge">{allProviders.length}</span>
        </button>
        <button
          className={`tab-btn${tab === 'repos' ? ' active' : ''}`}
          onClick={() => setTab('repos')}
        >
          Repositories
        </button>
      </div>

      <div style={{ marginTop: 20 }}>
        {tab === 'plugins' && <PluginsTab />}
        {tab === 'repos' && <RepositoriesTab enabledIds={enabledIds} allProviders={allProviders} />}
      </div>
    </div>
  );
}
