import { useState, useEffect } from 'react';
import { Navigate, Route, Routes, NavLink } from 'react-router-dom';
import DashboardPage from './pages/DashboardPage';
import ClustersPage from './pages/ClustersPage';
import ReleasesPage from './pages/ReleasesPage';
import LoginPage from './pages/LoginPage';
import AuditPage from './pages/AuditPage';
import NotificationsPage from './pages/NotificationsPage';
import ReportsPage from './pages/ReportsPage';
import SettingsPage from './pages/SettingsPage';
import RepositoriesPage from './pages/RepositoriesPage';
import { useStatusStream } from './lib/sse/useStatusStream';

type Theme = 'dark' | 'light' | 'warm';

const THEMES: { mode: Theme; title: string }[] = [
  { mode: 'dark',  title: 'Dark'  },
  { mode: 'light', title: 'Light' },
  { mode: 'warm',  title: 'Warm'  },
];

function useTheme(): [Theme, (t: Theme) => void] {
  const [theme, setThemeState] = useState<Theme>(
    () => (localStorage.getItem('theme') as Theme) ?? 'dark'
  );

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }, [theme]);

  return [theme, setThemeState];
}

function RequireAuth({ children }: { children: JSX.Element }) {
  const token = localStorage.getItem('auth_token');
  if (!token) return <Navigate to="/login" replace />;
  return children;
}

function NavItem({ to, icon, label }: { to: string; icon: string; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
    >
      <span>{icon}</span>
      <span>{label}</span>
    </NavLink>
  );
}

function Shell() {
  const token = localStorage.getItem('auth_token');
  const stream = useStatusStream(token);
  const [theme, setTheme] = useTheme();

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="sidebar-logo">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
          </svg>
          <h3>KubeHelm</h3>
        </div>
        <nav className="sidebar-nav">
          <NavItem to="/dashboard"      icon="⬡" label="Dashboard" />
          <NavItem to="/clusters"       icon="⊞" label="Clusters" />
          <NavItem to="/releases"       icon="⬛" label="Helm Releases" />
          <NavItem to="/audit"          icon="≡" label="Audit Log" />
          <NavItem to="/notifications"  icon="◎" label="Notifications" />
          <NavItem to="/reports"        icon="⊙" label="Reports" />
          <NavItem to="/repositories"    icon="⊗" label="Repositories" />
          <NavItem to="/settings"       icon="⚙" label="Settings" />
        </nav>
        <div className="sidebar-footer">
          <button
            className="mode-cycle-btn"
            title={`Switch mode (current: ${theme})`}
            onClick={() => {
              const idx = THEMES.findIndex(t => t.mode === theme);
              setTheme(THEMES[(idx + 1) % THEMES.length].mode);
            }}
          >
            <span className="mode-cycle-icon" data-mode={theme} />
            <span className="mode-cycle-label">Mode</span>
            <span className="mode-cycle-name">{theme.charAt(0).toUpperCase() + theme.slice(1)}</span>
          </button>
          <div className="flex items-center gap-2 text-sm text-muted" style={{ marginBottom: 8 }}>
            <span className={`sse-dot ${stream.connected ? 'connected' : ''}`} />
            <span>{stream.connected ? 'Live' : 'Offline'}</span>
          </div>
          <button
            className="btn btn-ghost btn-sm"
            style={{ width: '100%' }}
            onClick={() => {
              localStorage.removeItem('auth_token');
              window.location.href = '/login';
            }}
          >
            Sign out
          </button>
        </div>
      </aside>
      <main className="main">
        <Routes>
          <Route path="/dashboard"     element={<DashboardPage />} />
          <Route path="/clusters"      element={<ClustersPage />} />
          <Route path="/releases"      element={<ReleasesPage />} />
          <Route path="/audit"         element={<AuditPage />} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/reports"       element={<ReportsPage />} />
          <Route path="/repositories"  element={<RepositoriesPage />} />
          <Route path="/settings"      element={<SettingsPage />} />
          <Route path="*"              element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </main>
    </div>
  );
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <Shell />
          </RequireAuth>
        }
      />
    </Routes>
  );
}
