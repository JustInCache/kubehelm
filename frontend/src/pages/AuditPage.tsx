import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { auditApi } from '../lib/api/endpoints';
import { statusBadgeClass, formatDate } from '../lib/utils';

export default function AuditPage() {
  const [page, setPage] = useState(1);
  const limit = 50;

  const { data: events, isLoading, isError } = useQuery({
    queryKey: ['audit-events', page],
    queryFn: () => auditApi.listEvents({ page, limit }),
    refetchInterval: 15000,
  });

  const compliance = useQuery({
    queryKey: ['compliance'],
    queryFn: () => auditApi.compliance(),
  });

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Audit Log</div>
          <div className="page-subtitle">All actions performed on managed clusters</div>
        </div>
      </div>

      {compliance.data && compliance.data.length > 0 && (
        <div className="card" style={{ marginBottom: 20 }}>
          <div className="card-title">Compliance Checks</div>
          <div className="table-wrap">
            <table>
              <thead><tr><th>Check</th><th>Category</th><th>Status</th><th>Message</th></tr></thead>
              <tbody>
                {compliance.data.map(c => (
                  <tr key={c.id}>
                    <td className="col-name">{c.name}</td>
                    <td><span className="badge badge-blue">{c.category}</span></td>
                    <td><span className={`badge ${statusBadgeClass(c.status)}`}>{c.status}</span></td>
                    <td className="col-muted">{c.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="card" style={{ padding: 0 }}>
        <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
          <span className="fw-600">Audit Events</span>
        </div>
        {isLoading && <div className="loading">Loading events…</div>}
        {isError && <div className="alert alert-error" style={{ margin: 16 }}>Failed to load audit events</div>}
        {events && !events.length && (
          <div className="empty-state">
            <h3>No audit events yet</h3>
            <p>Events are recorded when clusters, releases, or settings are modified.</p>
          </div>
        )}
        {events && events.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>User</th><th>Action</th><th>Resource</th>
                  <th>Cluster</th><th>Namespace</th><th>When</th>
                </tr>
              </thead>
              <tbody>
                {events.map(e => (
                  <tr key={e.id}>
                    <td className="col-mono" style={{ fontSize: 12 }}>{e.username}</td>
                    <td><span className="badge badge-blue">{e.action}</span></td>
                    <td>
                      <span className="col-name">{e.resourceType}</span>
                      {e.resourceName && <span className="col-muted"> / {e.resourceName}</span>}
                    </td>
                    <td className="col-muted">{e.clusterName || '—'}</td>
                    <td className="col-muted">{e.namespace || '—'}</td>
                    <td className="col-muted">{formatDate(e.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex items-center justify-between" style={{ padding: '10px 16px', borderTop: '1px solid var(--border)' }}>
          <span className="text-muted text-sm">Page {page}</span>
          <div className="flex gap-2">
            <button className="btn btn-ghost btn-sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <button className="btn btn-ghost btn-sm" disabled={(events?.length ?? 0) < limit} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
        </div>
      </div>
    </div>
  );
}
