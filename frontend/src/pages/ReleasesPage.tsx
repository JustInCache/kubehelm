import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { releasesApi, clustersApi, repositoriesApi } from '../lib/api/endpoints';
import { statusBadgeClass, formatDate, formatDistanceToNow } from '../lib/utils';
import type { ChartInfo, HelmRelease, HelmRevision } from '../lib/api/types';

// ── Helpers ───────────────────────────────────────────────────────────────────

function apiError(e: unknown, fallback = 'Operation failed'): string {
  const err = e as { response?: { data?: { error?: string } } };
  return err?.response?.data?.error ?? fallback;
}

// ── Install Chart Modal ───────────────────────────────────────────────────────

function InstallModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [step, setStep] = useState<'target' | 'chart' | 'configure'>('target');
  const [clusterId, setClusterId] = useState('');
  const [namespace, setNamespace] = useState('default');
  const [repoId, setRepoId] = useState('');
  const [selectedChart, setSelectedChart] = useState<ChartInfo | null>(null);
  const [releaseName, setReleaseName] = useState('');
  const [version, setVersion] = useState('');
  const [valuesYaml, setValuesYaml] = useState('');
  const [error, setError] = useState('');
  const [output, setOutput] = useState('');
  const [chartSearch, setChartSearch] = useState('');

  const clusters = useQuery({ queryKey: ['clusters'], queryFn: clustersApi.list });
  const repos = useQuery({ queryKey: ['helm-repos'], queryFn: repositoriesApi.list });
  const namespaceQuery = useQuery({
    queryKey: ['namespaces', clusterId],
    queryFn: () => clustersApi.namespaces(clusterId),
    enabled: !!clusterId,
  });

  const chartsQuery = useQuery({
    queryKey: ['helm-charts', repoId],
    queryFn: () => releasesApi.listCharts(repoId),
    enabled: !!repoId,
  });

  // Group charts by name with all versions
  const chartMap = new Map<string, ChartInfo[]>();
  chartsQuery.data?.forEach(c => {
    if (!chartMap.has(c.name)) chartMap.set(c.name, []);
    chartMap.get(c.name)!.push(c);
  });
  // Sort versions descending within each chart
  chartMap.forEach(versions => versions.sort((a, b) => b.version.localeCompare(a.version, undefined, { numeric: true })));

  const filteredCharts = [...chartMap.entries()].filter(([name]) =>
    !chartSearch || name.toLowerCase().includes(chartSearch.toLowerCase())
  );

  const installMut = useMutation({
    mutationFn: () => {
      // Parse simple key=value lines from valuesYaml textarea into a map
      const values: Record<string, string> = {};
      valuesYaml.split('\n').forEach(line => {
        const m = line.match(/^([^=]+)=(.*)$/);
        if (m) values[m[1].trim()] = m[2].trim();
      });
      return releasesApi.install({
        clusterId,
        namespace,
        releaseName,
        chart: `${selectedChart!.repoName}/${selectedChart!.name}`,
        version,
        values: Object.keys(values).length ? values : undefined,
      });
    },
    onSuccess: (data) => {
      setOutput(data.output || 'Installed successfully');
      qc.invalidateQueries({ queryKey: ['releases'] });
      qc.invalidateQueries({ queryKey: ['releases-all-ns'] });
    },
    onError: (e) => setError(apiError(e, 'Install failed')),
  });

  const handleSelectChart = (chart: ChartInfo) => {
    setSelectedChart(chart);
    setVersion(chart.version);
    if (!releaseName) setReleaseName(chart.name);
    setStep('configure');
  };

  return (
    <div className="modal-overlay" onClick={e => e.target === e.currentTarget && onClose()}>
      <div className="modal" style={{ maxWidth: 620 }}>
        <div className="modal-header">
          <span className="modal-title">
            {step === 'target' && 'Install Chart — Select Target'}
            {step === 'chart' && 'Install Chart — Pick Chart'}
            {step === 'configure' && `Configure: ${selectedChart?.repoName}/${selectedChart?.name}`}
          </span>
          <button className="btn btn-ghost btn-icon" onClick={onClose}>✕</button>
        </div>

        <div className="modal-body">
          {/* Step indicator */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
            {(['target', 'chart', 'configure'] as const).map((s, i) => (
              <div key={s} style={{ display: 'flex', alignItems: 'center', gap: 6, opacity: step === s ? 1 : 0.4, fontSize: 12 }}>
                <div style={{ width: 20, height: 20, borderRadius: '50%', background: step === s ? 'var(--accent)' : 'var(--bg3)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, fontWeight: 700 }}>{i + 1}</div>
                <span style={{ textTransform: 'capitalize' }}>{s}</span>
                {i < 2 && <span style={{ opacity: 0.3 }}>›</span>}
              </div>
            ))}
          </div>

          {/* ── Step 1: Target ── */}
          {step === 'target' && (
            <>
              <div className="form-group">
                <label className="form-label">Cluster <span style={{ color: 'var(--danger)' }}>*</span></label>
                <select className="form-select" value={clusterId} onChange={e => setClusterId(e.target.value)}>
                  <option value="">Select cluster…</option>
                  {clusters.data?.map(c => <option key={c.id} value={c.id}>{c.name} ({c.environment})</option>)}
                </select>
              </div>
              <div className="form-group">
                <label className="form-label">Namespace <span style={{ color: 'var(--danger)' }}>*</span></label>
                {namespaceQuery.data?.length ? (
                  <select className="form-select" value={namespace} onChange={e => setNamespace(e.target.value)}>
                    {namespaceQuery.data.map(ns => <option key={ns} value={ns}>{ns}</option>)}
                    <option value="__custom">Custom…</option>
                  </select>
                ) : (
                  <input className="form-input" value={namespace} onChange={e => setNamespace(e.target.value)} placeholder="default" />
                )}
                {namespace === '__custom' && (
                  <input className="form-input" style={{ marginTop: 6 }} placeholder="Enter namespace name" onChange={e => setNamespace(e.target.value)} />
                )}
              </div>
              <div className="form-group">
                <label className="form-label">Repository <span style={{ color: 'var(--danger)' }}>*</span></label>
                <select className="form-select" value={repoId} onChange={e => setRepoId(e.target.value)}>
                  <option value="">Select repository…</option>
                  {repos.data?.filter(r => r.url && r.status === 'ok').map(r => (
                    <option key={r.id} value={r.id}>{r.name} ({r.providerId})</option>
                  ))}
                </select>
                {repos.data?.length === 0 && (
                  <div className="form-hint" style={{ color: 'var(--warning)', marginTop: 4 }}>
                    No repositories configured. Add one in the Repositories page first.
                  </div>
                )}
              </div>
            </>
          )}

          {/* ── Step 2: Pick Chart ── */}
          {step === 'chart' && (
            <>
              <input
                className="form-input"
                placeholder="Search charts…"
                value={chartSearch}
                onChange={e => setChartSearch(e.target.value)}
                style={{ marginBottom: 12 }}
              />
              {chartsQuery.isLoading && <div className="text-muted text-sm">Loading charts…</div>}
              {chartsQuery.isError && <div className="alert alert-error">Failed to load charts — check that the repository URL is reachable and returns a valid index.yaml.</div>}
              <div style={{ maxHeight: 340, overflowY: 'auto' }}>
                {filteredCharts.map(([name, versions]) => (
                  <div
                    key={name}
                    style={{ padding: '10px 12px', borderRadius: 6, marginBottom: 6, background: 'var(--bg2)', cursor: 'pointer', border: '1px solid transparent' }}
                    className="chart-card"
                    onClick={() => handleSelectChart(versions[0])}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
                      <div>
                        <div style={{ fontWeight: 600, fontSize: 14 }}>{name}</div>
                        {versions[0].description && (
                          <div className="text-muted" style={{ fontSize: 12, marginTop: 2, maxWidth: 380 }}>{versions[0].description}</div>
                        )}
                      </div>
                      <div style={{ textAlign: 'right', flexShrink: 0, marginLeft: 12 }}>
                        <span className="badge badge-blue">{versions[0].version}</span>
                        {versions[0].appVersion && <div className="text-muted" style={{ fontSize: 11, marginTop: 2 }}>app {versions[0].appVersion}</div>}
                        <div className="text-muted" style={{ fontSize: 11 }}>{versions.length} version{versions.length !== 1 ? 's' : ''}</div>
                      </div>
                    </div>
                  </div>
                ))}
                {!chartsQuery.isLoading && filteredCharts.length === 0 && (
                  <div className="text-muted text-sm">No charts found.</div>
                )}
              </div>
            </>
          )}

          {/* ── Step 3: Configure ── */}
          {step === 'configure' && selectedChart && (
            <>
              {error && <div className="alert alert-error" style={{ marginBottom: 12 }}>{error}</div>}
              {output && <div className="alert alert-success" style={{ marginBottom: 12, whiteSpace: 'pre-wrap', fontFamily: 'monospace', fontSize: 12 }}>{output}</div>}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div className="form-group">
                  <label className="form-label">Release Name <span style={{ color: 'var(--danger)' }}>*</span></label>
                  <input className="form-input" value={releaseName} onChange={e => setReleaseName(e.target.value)} placeholder={selectedChart.name} />
                </div>
                <div className="form-group">
                  <label className="form-label">Version</label>
                  <input className="form-input" value={version} onChange={e => setVersion(e.target.value)} placeholder={selectedChart.version} />
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 12 }}>
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label">Cluster</label>
                  <div className="form-input" style={{ background: 'var(--bg3)', color: 'var(--text2)', cursor: 'default' }}>
                    {clusters.data?.find(c => c.id === clusterId)?.name}
                  </div>
                </div>
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label">Namespace</label>
                  <div className="form-input" style={{ background: 'var(--bg3)', color: 'var(--text2)', cursor: 'default' }}>
                    {namespace}
                  </div>
                </div>
              </div>
              <div className="form-group">
                <label className="form-label">Values <span className="text-muted" style={{ fontWeight: 400, fontSize: 12 }}>(key=value per line, optional)</span></label>
                <textarea
                  className="form-input"
                  rows={5}
                  style={{ fontFamily: 'monospace', fontSize: 12, resize: 'vertical' }}
                  value={valuesYaml}
                  onChange={e => setValuesYaml(e.target.value)}
                  placeholder={'replicaCount=2\nimage.tag=1.2.3\nservice.type=ClusterIP'}
                />
              </div>
            </>
          )}
        </div>

        <div className="modal-footer">
          {step !== 'target' && !output && (
            <button className="btn btn-ghost" onClick={() => {
              if (step === 'configure') setStep('chart');
              else if (step === 'chart') setStep('target');
              setError('');
            }}>← Back</button>
          )}
          <button className="btn btn-ghost" onClick={onClose}>{output ? 'Close' : 'Cancel'}</button>
          {step === 'target' && (
            <button
              className="btn btn-primary"
              disabled={!clusterId || !namespace || !repoId}
              onClick={() => setStep('chart')}
            >
              Next: Pick Chart →
            </button>
          )}
          {step === 'configure' && !output && (
            <button
              className="btn btn-primary"
              disabled={!releaseName || installMut.isPending}
              onClick={() => { setError(''); installMut.mutate(); }}
            >
              {installMut.isPending ? '⟳ Installing…' : '⬇ Install'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Release Detail Drawer ─────────────────────────────────────────────────────

type DetailTab = 'overview' | 'values' | 'history' | 'upgrade' | 'manifest';

function ReleaseDetailDrawer({ release, onClose }: { release: HelmRelease; onClose: () => void }) {
  const qc = useQueryClient();
  const [tab, setTab] = useState<DetailTab>('overview');
  const [upgradeVersion, setUpgradeVersion] = useState(release.chartVersion);
  const [upgradeChart, setUpgradeChart] = useState(`${release.chartName}`);
  const [upgradeOutput, setUpgradeOutput] = useState('');
  const [upgradeError, setUpgradeError] = useState('');
  const [rollbackRevision, setRollbackRevision] = useState(0);
  const [rollbackOutput, setRollbackOutput] = useState('');
  const [rollbackError, setRollbackError] = useState('');
  const [uninstallOutput, setUninstallOutput] = useState('');
  const [uninstallError, setUninstallError] = useState('');

  const history = useQuery({
    queryKey: ['release-history', release.id],
    queryFn: () => releasesApi.history(release.id),
    enabled: tab === 'history' || tab === 'overview',
  });

  const values = useQuery({
    queryKey: ['release-values', release.id],
    queryFn: () => releasesApi.values(release.id),
    enabled: tab === 'values',
  });

  const [selectedRev, setSelectedRev] = useState<HelmRevision | null>(null);
  const manifest = useQuery({
    queryKey: ['manifest', release.id, selectedRev?.revision ?? 0],
    queryFn: () => releasesApi.manifest(release.id, selectedRev?.revision ?? release.revision),
    enabled: tab === 'manifest',
  });

  useEffect(() => {
    if (tab === 'manifest' && !selectedRev && history.data?.length) {
      setSelectedRev(history.data.find(r => r.revision === release.revision) ?? history.data[0]);
    }
  }, [tab, history.data, selectedRev, release.revision]);

  const upgradeMut = useMutation({
    mutationFn: () => releasesApi.upgrade(release.id, { chart: upgradeChart, version: upgradeVersion }),
    onSuccess: (data) => {
      setUpgradeOutput(data.output || data.message || 'Upgraded successfully');
      qc.invalidateQueries({ queryKey: ['releases'] });
      qc.invalidateQueries({ queryKey: ['releases-all-ns'] });
      qc.invalidateQueries({ queryKey: ['release-history', release.id] });
    },
    onError: (e) => setUpgradeError(apiError(e, 'Upgrade failed')),
  });

  const rollbackMut = useMutation({
    mutationFn: () => releasesApi.rollback(release.id, rollbackRevision),
    onSuccess: (data) => {
      setRollbackOutput(data.output || data.message || 'Rolled back successfully');
      qc.invalidateQueries({ queryKey: ['releases'] });
      qc.invalidateQueries({ queryKey: ['release-history', release.id] });
    },
    onError: (e) => setRollbackError(apiError(e, 'Rollback failed')),
  });

  const uninstallMut = useMutation({
    mutationFn: () => releasesApi.uninstall(release.id),
    onSuccess: (data) => {
      setUninstallOutput(data.output || 'Uninstalled successfully');
      qc.invalidateQueries({ queryKey: ['releases'] });
      qc.invalidateQueries({ queryKey: ['releases-all-ns'] });
    },
    onError: (e) => setUninstallError(apiError(e, 'Uninstall failed')),
  });

  const prevRevisions = history.data?.filter(r => r.revision < release.revision) ?? [];

  return (
    <div style={{
      position: 'fixed', top: 0, right: 0, bottom: 0, width: 680,
      background: 'var(--bg1)', borderLeft: '1px solid var(--border)',
      zIndex: 300, display: 'flex', flexDirection: 'column',
      boxShadow: '-4px 0 24px rgba(0,0,0,0.3)',
    }}>
      {/* Header */}
      <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontWeight: 700, fontSize: 18 }}>{release.name}</div>
          <div style={{ display: 'flex', gap: 8, marginTop: 4, flexWrap: 'wrap' }}>
            <span className="badge badge-blue">{release.namespace}</span>
            <span className={`badge ${statusBadgeClass(release.status)}`}>{release.status}</span>
            <span className="badge" style={{ background: 'var(--bg3)', color: 'var(--text2)' }}>{release.clusterName}</span>
            <span className="badge" style={{ background: 'var(--bg3)', color: 'var(--text2)' }}>#{release.revision}</span>
          </div>
        </div>
        <button className="btn btn-ghost btn-icon" onClick={onClose}>✕</button>
      </div>

      {/* Tabs */}
      <div style={{ display: 'flex', borderBottom: '1px solid var(--border)', padding: '0 20px' }}>
        {(['overview', 'values', 'history', 'upgrade', 'manifest'] as DetailTab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              padding: '10px 14px', border: 'none', background: 'none', cursor: 'pointer',
              fontSize: 13, fontWeight: tab === t ? 600 : 400,
              color: tab === t ? 'var(--accent)' : 'var(--text2)',
              borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
              textTransform: 'capitalize',
            }}
          >{t}</button>
        ))}
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 20 }}>

        {/* ── Overview ── */}
        {tab === 'overview' && (
          <div>
            <div className="card" style={{ marginBottom: 16 }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px 24px' }}>
                {[
                  ['Chart', `${release.chartName}-${release.chartVersion}`],
                  ['App Version', release.appVersion || '—'],
                  ['Namespace', release.namespace],
                  ['Cluster', release.clusterName],
                  ['Revision', `#${release.revision}`],
                  ['Last Updated', formatDate(release.updatedAt || release.createdAt)],
                ].map(([label, value]) => (
                  <div key={label}>
                    <div className="text-muted" style={{ fontSize: 11, marginBottom: 2 }}>{label}</div>
                    <div style={{ fontSize: 13, fontWeight: 500 }}>{value}</div>
                  </div>
                ))}
              </div>
            </div>

            {/* Recent history */}
            <div className="fw-600 text-sm" style={{ marginBottom: 8 }}>Recent History</div>
            {history.isLoading && <div className="text-muted text-sm">Loading…</div>}
            <ul className="timeline">
              {history.data?.slice(0, 5).map(r => (
                <li key={r.revision} className={`timeline-item ${r.revision === release.revision ? 'active' : ''}`}>
                  <div className="fw-600 text-sm">
                    #{r.revision} — {r.chartVersion}
                    {r.revision === release.revision && <span className="badge badge-green" style={{ marginLeft: 6, fontSize: 10 }}>current</span>}
                  </div>
                  <div className="text-muted" style={{ fontSize: 12 }}>{r.description || r.status} · {formatDistanceToNow(r.deployedAt)}</div>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* ── Values ── */}
        {tab === 'values' && (
          <div>
            <div className="fw-600 text-sm" style={{ marginBottom: 8 }}>User-supplied values for revision #{release.revision}</div>
            {values.isLoading && <div className="text-muted text-sm">Loading values…</div>}
            {values.isError && <div className="alert alert-error">Could not fetch values — helm CLI may not be available.</div>}
            {values.data && (
              <div className="code-block" style={{ whiteSpace: 'pre', fontSize: 12, maxHeight: 500, overflowY: 'auto' }}>
                {values.data.values || '# No user-supplied values (using chart defaults)'}
              </div>
            )}
          </div>
        )}

        {/* ── History ── */}
        {tab === 'history' && (
          <div>
            {history.isLoading && <div className="text-muted text-sm">Loading…</div>}
            <ul className="timeline">
              {history.data?.map(r => (
                <li key={r.revision} className={`timeline-item ${r.revision === release.revision ? 'active' : ''}`}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span className="fw-600 text-sm">#{r.revision}</span>
                    <span className="badge badge-blue">{r.chartVersion}</span>
                    <span className={`badge ${statusBadgeClass(r.status)}`}>{r.status}</span>
                    {r.revision === release.revision && <span className="badge badge-green">current</span>}
                  </div>
                  <div className="text-muted" style={{ fontSize: 12, marginTop: 2 }}>
                    {formatDate(r.deployedAt)} · {r.description}
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* ── Upgrade ── */}
        {tab === 'upgrade' && (
          <div>
            <div className="card" style={{ marginBottom: 16 }}>
              <div className="fw-600 text-sm" style={{ marginBottom: 14 }}>Upgrade Release</div>
              {upgradeError && <div className="alert alert-error" style={{ marginBottom: 12 }}>{upgradeError}</div>}
              {upgradeOutput && <div className="alert alert-success" style={{ marginBottom: 12, whiteSpace: 'pre-wrap', fontFamily: 'monospace', fontSize: 12 }}>{upgradeOutput}</div>}
              <div className="form-group">
                <label className="form-label">Chart (repo/chart)</label>
                <input className="form-input" value={upgradeChart} onChange={e => setUpgradeChart(e.target.value)} placeholder={release.chartName} />
                <div className="form-hint text-muted" style={{ fontSize: 11, marginTop: 4 }}>Format: repo-name/chart-name or OCI URI</div>
              </div>
              <div className="form-group">
                <label className="form-label">Version</label>
                <input className="form-input" value={upgradeVersion} onChange={e => setUpgradeVersion(e.target.value)} placeholder="latest" />
              </div>
              <button
                className="btn btn-primary"
                disabled={upgradeMut.isPending || !!upgradeOutput}
                onClick={() => { setUpgradeError(''); upgradeMut.mutate(); }}
              >
                {upgradeMut.isPending ? '⟳ Upgrading…' : '⬆ Upgrade'}
              </button>
            </div>

            <div className="card">
              <div className="fw-600 text-sm" style={{ marginBottom: 14 }}>Rollback to Previous Revision</div>
              {rollbackError && <div className="alert alert-error" style={{ marginBottom: 12 }}>{rollbackError}</div>}
              {rollbackOutput && <div className="alert alert-success" style={{ marginBottom: 12 }}>{rollbackOutput}</div>}
              {prevRevisions.length === 0 && (
                <div className="text-muted text-sm">No previous revisions available.</div>
              )}
              {prevRevisions.length > 0 && (
                <>
                  <div className="form-group">
                    <label className="form-label">Roll back to</label>
                    <select className="form-select" value={rollbackRevision} onChange={e => setRollbackRevision(Number(e.target.value))}>
                      <option value={0}>Select revision…</option>
                      {prevRevisions.map(r => (
                        <option key={r.revision} value={r.revision}>
                          #{r.revision} — {r.chartVersion} ({formatDistanceToNow(r.deployedAt)})
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    className="btn btn-warning btn-sm"
                    disabled={!rollbackRevision || rollbackMut.isPending || !!rollbackOutput}
                    onClick={() => { setRollbackError(''); rollbackMut.mutate(); }}
                  >
                    {rollbackMut.isPending ? '⟳ Rolling back…' : '↩ Rollback'}
                  </button>
                </>
              )}
            </div>

            {/* Danger zone */}
            <div style={{ marginTop: 16, padding: '14px 16px', borderRadius: 8, border: '1px solid var(--danger)', background: 'rgba(239,68,68,0.05)' }}>
              <div className="fw-600 text-sm" style={{ color: 'var(--danger)', marginBottom: 8 }}>Danger Zone</div>
              {uninstallError && <div className="alert alert-error" style={{ marginBottom: 8 }}>{uninstallError}</div>}
              {uninstallOutput && <div className="alert alert-success" style={{ marginBottom: 8 }}>{uninstallOutput}</div>}
              <p className="text-muted" style={{ fontSize: 12, marginBottom: 10 }}>
                Uninstalling <strong>{release.name}</strong> will remove all Kubernetes resources managed by this release. This cannot be undone.
              </p>
              <button
                className="btn btn-danger btn-sm"
                disabled={uninstallMut.isPending || !!uninstallOutput}
                onClick={() => {
                  if (confirm(`Uninstall "${release.name}" from namespace "${release.namespace}"? This cannot be undone.`)) {
                    setUninstallError('');
                    uninstallMut.mutate();
                  }
                }}
              >
                {uninstallMut.isPending ? '⟳ Uninstalling…' : '🗑 Uninstall Release'}
              </button>
            </div>
          </div>
        )}

        {/* ── Manifest ── */}
        {tab === 'manifest' && (
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
              <label className="form-label" style={{ margin: 0 }}>Revision</label>
              <select className="form-select" style={{ width: 180 }} value={selectedRev?.revision ?? ''} onChange={e => {
                const rev = history.data?.find(r => r.revision === Number(e.target.value));
                if (rev) setSelectedRev(rev);
              }}>
                {history.data?.map(r => (
                  <option key={r.revision} value={r.revision}>#{r.revision} — {r.chartVersion}</option>
                ))}
              </select>
            </div>
            {manifest.isLoading && <div className="text-muted text-sm">Loading manifest…</div>}
            {manifest.isError && <div className="alert alert-error">Could not fetch manifest.</div>}
            {manifest.data && (
              <div className="code-block" style={{ whiteSpace: 'pre', fontSize: 11, maxHeight: 520, overflowY: 'auto' }}>
                {manifest.data.manifest || '(empty manifest)'}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function ReleasesPage() {
  const [search, setSearch] = useState('');
  const [namespace, setNamespace] = useState('');
  const [clusterId, setClusterId] = useState('');
  const [showInstall, setShowInstall] = useState(false);
  const [detailRelease, setDetailRelease] = useState<HelmRelease | null>(null);

  const clusters = useQuery({ queryKey: ['clusters'], queryFn: () => clustersApi.list() });

  const { data: allReleases } = useQuery({
    queryKey: ['releases-all-ns'],
    queryFn: () => releasesApi.list({ limit: 500 }),
    refetchInterval: 60000,
    staleTime: 30000,
  });

  const { data: releases, isLoading, isError, refetch } = useQuery({
    queryKey: ['releases', namespace, search],
    queryFn: () => releasesApi.list({ namespace: namespace || undefined, search: search || undefined, limit: 500 }),
    refetchInterval: 30000,
  });

  const filtered = releases?.filter(r => !clusterId || r.clusterId === clusterId) ?? [];
  const namespaces = [...new Set(allReleases?.map(r => r.namespace) ?? [])].sort();

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Helm Releases</div>
          <div className="page-subtitle">
            {isLoading ? 'Loading…' : `${filtered.length} release${filtered.length !== 1 ? 's' : ''}`}
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="btn btn-ghost btn-sm" onClick={() => refetch()}>↻ Refresh</button>
          <button className="btn btn-primary btn-sm" onClick={() => setShowInstall(true)}>+ Install Chart</button>
        </div>
      </div>

      {showInstall && <InstallModal onClose={() => setShowInstall(false)} />}
      {detailRelease && (
        <>
          <div
            style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', zIndex: 299 }}
            onClick={() => setDetailRelease(null)}
          />
          <ReleaseDetailDrawer release={detailRelease} onClose={() => setDetailRelease(null)} />
        </>
      )}

      <div className="filters">
        <input
          className="search-input"
          placeholder="Search releases…"
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
        <select className="form-select" style={{ width: 160 }} value={clusterId} onChange={e => setClusterId(e.target.value)}>
          <option value="">All Clusters</option>
          {clusters.data?.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select className="form-select" style={{ width: 160 }} value={namespace} onChange={e => setNamespace(e.target.value)}>
          <option value="">All Namespaces</option>
          {namespaces.map(ns => <option key={ns} value={ns}>{ns}</option>)}
        </select>
      </div>

      {isError && <div className="alert alert-error">Failed to load releases</div>}

      {!isLoading && !filtered.length && (
        <div className="empty-state">
          <div className="empty-state-icon">⬛</div>
          <h3>No releases found</h3>
          <p>
            {!clusters.data?.length
              ? 'Add a Kubernetes cluster first to discover Helm releases.'
              : 'No Helm releases found. Install a chart to get started.'}
          </p>
          <button className="btn btn-primary" onClick={() => setShowInstall(true)}>+ Install Chart</button>
        </div>
      )}

      {filtered.length > 0 && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Release</th><th>Cluster</th><th>Namespace</th>
                  <th>Chart</th><th>App Version</th><th>Status</th>
                  <th>Revision</th><th>Updated</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map(r => (
                  <tr
                    key={r.id}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setDetailRelease(r)}
                  >
                    <td className="col-name" style={{ fontWeight: 600 }}>{r.name}</td>
                    <td className="col-muted">{r.clusterName}</td>
                    <td><span className="badge badge-blue">{r.namespace}</span></td>
                    <td className="col-mono">{r.chartName}-{r.chartVersion}</td>
                    <td className="col-muted">{r.appVersion || '—'}</td>
                    <td><span className={`badge ${statusBadgeClass(r.status)}`}>{r.status}</span></td>
                    <td className="text-muted">#{r.revision}</td>
                    <td className="col-muted">{formatDistanceToNow(r.updatedAt || r.createdAt)}</td>
                    <td onClick={e => e.stopPropagation()}>
                      <div className="flex gap-2">
                        <button
                          className="btn btn-ghost btn-sm"
                          onClick={() => { setDetailRelease(r); }}
                          title="View details"
                        >
                          Details
                        </button>
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => { setDetailRelease(r); }}
                          title="Upgrade release"
                        >
                          Upgrade
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
