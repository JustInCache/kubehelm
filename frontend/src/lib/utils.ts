export function formatDistanceToNow(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  });
}

export function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    deployed: 'badge-green', connected: 'badge-green', completed: 'badge-green', ok: 'badge-green',
    failed: 'badge-red', error: 'badge-red',
    pending: 'badge-yellow', pending_upgrade: 'badge-yellow', drifted: 'badge-yellow',
    superseded: 'badge-gray', uninstalled: 'badge-gray', unknown: 'badge-gray',
  };
  return map[status?.toLowerCase()] ?? 'badge-gray';
}
