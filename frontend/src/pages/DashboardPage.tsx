import { useQuery } from '@tanstack/react-query';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { clustersApi, releasesApi, auditApi } from '../lib/api/endpoints';
import { formatDistanceToNow } from '../lib/utils';

function statusBadge(status: string) {
  const map: Record<string, string> = {
    deployed: 'badge-green', failed: 'badge-red', pending: 'badge-yellow',
    superseded: 'badge-gray', uninstalled: 'badge-gray', connected: 'badge-green',
    error: 'badge-red',
  };
  return <span className={`badge ${map[status] ?? 'badge-gray'}`}>{status}</span>;
}

export default function DashboardPage() {
  const clusters = useQuery({ queryKey: ['clusters'], queryFn: () => clustersApi.list(), refetchInterval: 30000 });
  const releases = useQuery({ queryKey: ['releases'], queryFn: () => releasesApi.list({ limit: 500 }), refetchInterval: 30000 });
  const auditEvents = useQuery({ queryKey: ['audit-events'], queryFn: () => auditApi.listEvents({ limit: 10 }) });

  const clusterCount = clusters.data?.length ?? 0;
  const releaseCount = releases.data?.length ?? 0;
  const deployedCount = releases.data?.filter(r => r.status === 'deployed').length ?? 0;
  const failedCount = releases.data?.filter(r => r.status === 'failed').length ?? 0;

  const statusData = [
    { name: 'Deployed', value: deployedCount, color: '#3fb950' },
    { name: 'Failed', value: failedCount, color: '#f85149' },
    { name: 'Other', value: releaseCount - deployedCount - failedCount, color: '#8b949e' },
  ].filter(d => d.value > 0);

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Dashboard</div>
          <div className="page-subtitle">Platform overview and real-time status</div>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{clusterCount}</div>
          <div className="stat-label">Clusters</div>
          <div className="stat-delta">
            {clusters.data?.filter(c => c.status === 'connected').length ?? 0} connected
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{releaseCount}</div>
          <div className="stat-label">Helm Releases</div>
          <div className="stat-delta">{deployedCount} deployed</div>
        </div>
        <div className="stat-card">
          <div className="stat-value" style={{ color: failedCount > 0 ? 'var(--danger)' : 'var(--success)' }}>
            {failedCount}
          </div>
          <div className="stat-label">Failed Releases</div>
          {failedCount > 0 && <div className="stat-delta" style={{ color: 'var(--danger)' }}>Needs attention</div>}
        </div>
        <div className="stat-card">
          <div className="stat-value">{auditEvents.data?.length ?? 0}</div>
          <div className="stat-label">Recent Actions</div>
          <div className="stat-delta">last 24h</div>
        </div>
      </div>

      <div className="grid-2" style={{ marginBottom: 20 }}>
        <div className="card">
          <div className="card-title">Release Status Distribution</div>
          {releases.isLoading
            ? <div className="loading">Loading…</div>
            : statusData.length === 0
              ? <div className="text-muted text-sm">No releases yet</div>
              : (
                <ResponsiveContainer width="100%" height={180}>
                  <BarChart data={statusData} barSize={40}>
                    <XAxis dataKey="name" stroke="#6e7681" tick={{ fontSize: 12 }} />
                    <YAxis stroke="#6e7681" tick={{ fontSize: 12 }} />
                    <Tooltip
                      contentStyle={{ background: '#161b22', border: '1px solid #30363d', borderRadius: 6, fontSize: 12 }}
                    />
                    <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                      {statusData.map((entry, i) => <Cell key={i} fill={entry.color} />)}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              )
          }
        </div>

        <div className="card">
          <div className="card-title">Clusters</div>
          {clusters.isLoading
            ? <div className="loading">Loading…</div>
            : !clusters.data?.length
              ? <div className="empty-state" style={{ padding: '24px 0' }}>
                  <div>No clusters registered.</div>
                  <a href="/clusters" style={{ color: 'var(--accent)', fontSize: 13 }}>Add your first cluster →</a>
                </div>
              : (
                <table>
                  <thead>
                    <tr><th>Name</th><th>Env</th><th>Status</th><th>Releases</th></tr>
                  </thead>
                  <tbody>
                    {clusters.data.slice(0, 5).map(c => (
                      <tr key={c.id}>
                        <td className="col-name">{c.name}</td>
                        <td><span className="badge badge-blue">{c.environment}</span></td>
                        <td>{statusBadge(c.status)}</td>
                        <td className="text-muted">{c.releaseCount ?? 0}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )
          }
        </div>
      </div>

      <div className="card">
        <div className="card-title">Recent Audit Events</div>
        {auditEvents.isLoading
          ? <div className="loading">Loading…</div>
          : !auditEvents.data?.length
            ? <div className="text-muted text-sm">No recent events</div>
            : (
              <table>
                <thead>
                  <tr><th>User</th><th>Action</th><th>Resource</th><th>When</th></tr>
                </thead>
                <tbody>
                  {auditEvents.data.slice(0, 8).map(e => (
                    <tr key={e.id}>
                      <td className="col-muted">{e.username}</td>
                      <td><span className="badge badge-blue">{e.action}</span></td>
                      <td><span className="text-mono">{e.resourceType}/{e.resourceName}</span></td>
                      <td className="col-muted">{formatDistanceToNow(e.createdAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )
        }
      </div>
    </div>
  );
}
