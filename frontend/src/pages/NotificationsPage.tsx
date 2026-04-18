import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { notificationsApi } from '../lib/api/endpoints';
import { formatDate } from '../lib/utils';

const CHANNEL_TYPES = ['slack', 'email', 'webhook', 'pagerduty', 'teams'];
const EVENT_TYPES = [
  'drift.detected', 'release.deployed', 'release.failed',
  'release.rollback', 'cluster.error', 'approval.required',
];

export default function NotificationsPage() {
  const qc = useQueryClient();
  const [showAddChannel, setShowAddChannel] = useState(false);
  const [showAddRule, setShowAddRule] = useState(false);
  const [channelName, setChannelName] = useState('');
  const [channelType, setChannelType] = useState('slack');
  const [ruleName, setRuleName] = useState('');
  const [ruleEvents, setRuleEvents] = useState<string[]>([]);
  const [ruleChannels, setRuleChannels] = useState<string[]>([]);
  const [error, setError] = useState('');

  const channels = useQuery({ queryKey: ['channels'], queryFn: () => notificationsApi.listChannels() });
  const rules = useQuery({ queryKey: ['rules'], queryFn: () => notificationsApi.listRules() });

  const createChannelMut = useMutation({
    mutationFn: () => notificationsApi.createChannel({ name: channelName, type: channelType }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['channels'] }); setShowAddChannel(false); setChannelName(''); },
    onError: () => setError('Failed to create channel'),
  });

  const deleteChannelMut = useMutation({
    mutationFn: (id: string) => notificationsApi.deleteChannel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['channels'] }),
  });

  const toggleChannelMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => notificationsApi.toggleChannel(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['channels'] }),
  });

  const createRuleMut = useMutation({
    mutationFn: () => notificationsApi.createRule({ name: ruleName, events: ruleEvents, channelIds: ruleChannels }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['rules'] }); setShowAddRule(false); setRuleName(''); setRuleEvents([]); setRuleChannels([]); },
    onError: () => setError('Failed to create rule'),
  });

  const toggleEvent = (e: string) => setRuleEvents(prev => prev.includes(e) ? prev.filter(x => x !== e) : [...prev, e]);
  const toggleChannel = (id: string) => setRuleChannels(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);

  return (
    <div>
      <div className="page-header">
        <div><div className="page-title">Notifications</div></div>
      </div>

      {error && <div className="alert alert-error">{error}</div>}

      {/* Channels */}
      <div className="card" style={{ marginBottom: 20 }}>
        <div className="flex items-center justify-between mb-3">
          <div className="card-title" style={{ margin: 0 }}>Channels</div>
          <button className="btn btn-primary btn-sm" onClick={() => setShowAddChannel(true)}>+ Add Channel</button>
        </div>

        {showAddChannel && (
          <div className="card" style={{ marginBottom: 12, background: 'var(--bg3)' }}>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Name</label>
                <input className="form-input" value={channelName} onChange={e => setChannelName(e.target.value)} placeholder="Ops Slack" />
              </div>
              <div className="form-group">
                <label className="form-label">Type</label>
                <select className="form-select" value={channelType} onChange={e => setChannelType(e.target.value)}>
                  {CHANNEL_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary btn-sm" onClick={() => createChannelMut.mutate()} disabled={!channelName || createChannelMut.isPending}>
                {createChannelMut.isPending ? 'Saving…' : 'Save'}
              </button>
              <button className="btn btn-ghost btn-sm" onClick={() => setShowAddChannel(false)}>Cancel</button>
            </div>
          </div>
        )}

        {channels.isLoading && <div className="text-muted text-sm">Loading…</div>}
        {!channels.isLoading && !channels.data?.length && (
          <div className="text-muted text-sm">No channels configured. Add one to receive notifications.</div>
        )}
        {channels.data?.map(ch => (
          <div key={ch.id} className="flex items-center justify-between" style={{ padding: '8px 0', borderBottom: '1px solid var(--border2)' }}>
            <div>
              <span className="fw-600">{ch.name}</span>
              <span className="badge badge-blue" style={{ marginLeft: 8 }}>{ch.type}</span>
              {ch.enabled
                ? <span className="badge badge-green" style={{ marginLeft: 6 }}>enabled</span>
                : <span className="badge badge-gray" style={{ marginLeft: 6 }}>disabled</span>}
            </div>
            <div className="flex gap-2">
              <button className="btn btn-ghost btn-sm" onClick={() => toggleChannelMut.mutate({ id: ch.id, enabled: !ch.enabled })}>
                {ch.enabled ? 'Disable' : 'Enable'}
              </button>
              <button className="btn btn-danger btn-sm" onClick={() => {
                if (window.confirm('Delete this channel?')) deleteChannelMut.mutate(ch.id);
              }}>Delete</button>
            </div>
          </div>
        ))}
      </div>

      {/* Rules */}
      <div className="card">
        <div className="flex items-center justify-between mb-3">
          <div className="card-title" style={{ margin: 0 }}>Notification Rules</div>
          <button className="btn btn-primary btn-sm" onClick={() => setShowAddRule(true)}>+ Add Rule</button>
        </div>

        {showAddRule && (
          <div className="card" style={{ marginBottom: 12, background: 'var(--bg3)' }}>
            <div className="form-group">
              <label className="form-label">Rule Name</label>
              <input className="form-input" value={ruleName} onChange={e => setRuleName(e.target.value)} placeholder="Drift Alerts" />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Events</label>
                {EVENT_TYPES.map(ev => (
                  <label key={ev} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, cursor: 'pointer', fontSize: 13 }}>
                    <input type="checkbox" checked={ruleEvents.includes(ev)} onChange={() => toggleEvent(ev)} />
                    {ev}
                  </label>
                ))}
              </div>
              <div className="form-group">
                <label className="form-label">Send to Channels</label>
                {channels.data?.map(ch => (
                  <label key={ch.id} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, cursor: 'pointer', fontSize: 13 }}>
                    <input type="checkbox" checked={ruleChannels.includes(ch.id)} onChange={() => toggleChannel(ch.id)} />
                    {ch.name}
                  </label>
                ))}
                {!channels.data?.length && <div className="text-muted text-sm">No channels yet</div>}
              </div>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary btn-sm" onClick={() => createRuleMut.mutate()} disabled={!ruleName || createRuleMut.isPending}>
                {createRuleMut.isPending ? 'Saving…' : 'Save Rule'}
              </button>
              <button className="btn btn-ghost btn-sm" onClick={() => setShowAddRule(false)}>Cancel</button>
            </div>
          </div>
        )}

        {rules.isLoading && <div className="text-muted text-sm">Loading…</div>}
        {!rules.isLoading && !rules.data?.length && (
          <div className="text-muted text-sm">No rules configured. Rules define when and where notifications are sent.</div>
        )}
        {rules.data?.map(rule => (
          <div key={rule.id} style={{ padding: '10px 0', borderBottom: '1px solid var(--border2)' }}>
            <div className="fw-600">{rule.name}</div>
            <div className="text-sm text-muted mt-2">
              Events: {rule.events.map(e => <span key={e} className="badge badge-blue" style={{ marginRight: 4 }}>{e}</span>)}
            </div>
            <div className="text-sm text-muted mt-2">Added {formatDate(rule.createdAt)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
