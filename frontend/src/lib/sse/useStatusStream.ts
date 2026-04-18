import { useEffect, useState } from 'react';

type StatusEvent = {
  type: string;
  timestamp: string;
  health: number;
};

export function useStatusStream(token: string | null) {
  const [event, setEvent] = useState<StatusEvent | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!token) return;
    const base = import.meta.env.VITE_API_ROOT || 'http://localhost:8080';
    const url = `${base}/api/stream?token=${encodeURIComponent(token)}`;
    const es = new EventSource(url, { withCredentials: false });

    es.addEventListener('connected', () => setConnected(true));
    es.addEventListener('status', (evt) => {
      try {
        setEvent(JSON.parse((evt as MessageEvent).data));
      } catch {
        // noop
      }
    });
    es.addEventListener('heartbeat', () => setConnected(true));
    es.onerror = () => setConnected(false);

    return () => es.close();
  }, [token]);

  return { connected, event };
}

