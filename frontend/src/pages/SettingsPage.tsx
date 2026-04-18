import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi } from '../lib/api/endpoints';
import type { User } from '../lib/api/types';

const ROLES = ['admin', 'sre', 'readonly'];

export default function SettingsPage() {
  const qc = useQueryClient();
  const [orgName, setOrgName] = useState('');
  const [orgSaved, setOrgSaved] = useState(false);
  const [error, setError] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: () => settingsApi.get(),
  });

  useEffect(() => {
    if (data?.organization?.name) {
      setOrgName(data.organization.name);
    }
  }, [data]);

  const updateOrgMut = useMutation({
    mutationFn: () => settingsApi.updateOrganization({ name: orgName }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['settings'] });
      setOrgSaved(true);
      setTimeout(() => setOrgSaved(false), 3000);
    },
    onError: () => setError('Failed to update organization'),
  });

  const updateRoleMut = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) => settingsApi.updateUserRole(userId, role),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
    onError: () => setError('Failed to update role'),
  });

  const currentUser = JSON.parse(localStorage.getItem('auth_user') || '{}') as User;

  if (isLoading) return <div className="loading">Loading…</div>;

  return (
    <div>
      <div className="page-header">
        <div><div className="page-title">Settings</div></div>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {orgSaved && <div className="alert alert-success">Organization settings saved.</div>}

      {/* Org Settings */}
      <div className="card" style={{ marginBottom: 20 }}>
        <div className="card-title">Organization</div>
        <div className="form-group" style={{ maxWidth: 360 }}>
          <label className="form-label">Organization Name</label>
          <input className="form-input" value={orgName} onChange={e => setOrgName(e.target.value)} />
        </div>
        <button
          className="btn btn-primary"
          onClick={() => updateOrgMut.mutate()}
          disabled={!orgName || updateOrgMut.isPending}
        >
          {updateOrgMut.isPending ? 'Saving…' : 'Save Changes'}
        </button>
      </div>

      {/* Current User */}
      <div className="card" style={{ marginBottom: 20 }}>
        <div className="card-title">My Account</div>
        <div className="text-sm">
          <div style={{ marginBottom: 6 }}><span className="text-muted">Name: </span>{currentUser.name}</div>
          <div style={{ marginBottom: 6 }}><span className="text-muted">Email: </span>{currentUser.email}</div>
          <div><span className="text-muted">Role: </span><span className="badge badge-purple">{currentUser.role}</span></div>
        </div>
      </div>

      {/* Users */}
      <div className="card">
        <div className="card-title">Team Members</div>
        {!data?.users?.length && <div className="text-muted text-sm">No users found.</div>}
        {data?.users && data.users.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead>
                <tr><th>Name</th><th>Email</th><th>Role</th><th>Actions</th></tr>
              </thead>
              <tbody>
                {data.users.map(u => (
                  <tr key={u.id}>
                    <td className="col-name">{u.name}</td>
                    <td className="col-mono" style={{ fontSize: 12 }}>{u.email}</td>
                    <td><span className="badge badge-purple">{u.role}</span></td>
                    <td>
                      {u.id !== currentUser.id && (
                        <select
                          className="form-select"
                          style={{ width: 130, padding: '4px 8px', fontSize: 12 }}
                          value={u.role}
                          onChange={e => updateRoleMut.mutate({ userId: u.id, role: e.target.value })}
                        >
                          {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
                        </select>
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
