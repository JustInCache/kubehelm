import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { reportsApi } from '../lib/api/endpoints';
import { statusBadgeClass, formatDate } from '../lib/utils';

const REPORT_TYPES = ['audit', 'drift', 'compliance', 'releases'];
const FORMATS = ['csv', 'json', 'pdf'];

export default function ReportsPage() {
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState('');
  const [type, setType] = useState('audit');
  const [format, setFormat] = useState('csv');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const reports = useQuery({
    queryKey: ['reports'],
    queryFn: () => reportsApi.list(),
    refetchInterval: 10000,
  });

  const createMut = useMutation({
    mutationFn: () => reportsApi.create({ name, type, format }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['reports'] });
      setShowCreate(false);
      setName('');
      setSuccess('Report generation started. It will be ready shortly.');
      setTimeout(() => setSuccess(''), 5000);
    },
    onError: () => setError('Failed to generate report'),
  });

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Reports</div>
          <div className="page-subtitle">Generate and download compliance and audit reports</div>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>+ Generate Report</button>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      {showCreate && (
        <div className="card" style={{ marginBottom: 20 }}>
          <div className="card-title">New Report</div>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Report Name</label>
              <input className="form-input" value={name} onChange={e => setName(e.target.value)} placeholder="Monthly Audit Report" />
            </div>
            <div className="form-group">
              <label className="form-label">Type</label>
              <select className="form-select" value={type} onChange={e => setType(e.target.value)}>
                {REPORT_TYPES.map(t => <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
              </select>
            </div>
          </div>
          <div className="form-group" style={{ maxWidth: 200 }}>
            <label className="form-label">Format</label>
            <select className="form-select" value={format} onChange={e => setFormat(e.target.value)}>
              {FORMATS.map(f => <option key={f} value={f}>{f.toUpperCase()}</option>)}
            </select>
          </div>
          <div className="flex gap-2">
            <button className="btn btn-primary" onClick={() => createMut.mutate()} disabled={!name || createMut.isPending}>
              {createMut.isPending ? '⟳ Generating…' : 'Generate'}
            </button>
            <button className="btn btn-ghost" onClick={() => setShowCreate(false)}>Cancel</button>
          </div>
        </div>
      )}

      <div className="card" style={{ padding: 0 }}>
        {reports.isLoading && <div className="loading">Loading reports…</div>}
        {!reports.isLoading && !reports.data?.length && (
          <div className="empty-state">
            <div className="empty-state-icon">⊙</div>
            <h3>No reports yet</h3>
            <p>Generate your first report to export audit or compliance data.</p>
          </div>
        )}
        {reports.data && reports.data.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead>
                <tr><th>Name</th><th>Type</th><th>Format</th><th>Status</th><th>Created</th><th>Actions</th></tr>
              </thead>
              <tbody>
                {reports.data.map(r => (
                  <tr key={r.id}>
                    <td className="col-name">{r.name}</td>
                    <td><span className="badge badge-purple">{r.type}</span></td>
                    <td><span className="badge badge-gray">{r.format.toUpperCase()}</span></td>
                    <td><span className={`badge ${statusBadgeClass(r.status)}`}>{r.status}</span></td>
                    <td className="col-muted">{formatDate(r.createdAt)}</td>
                    <td>
                      {r.status === 'completed' && (
                        <button className="btn btn-ghost btn-sm">↓ Download</button>
                      )}
                      {r.status === 'pending' && (
                        <span className="text-muted text-sm">⟳ Processing…</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
