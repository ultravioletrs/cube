import { Navigate, Routes, Route, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import AppLayout from '@/layouts/AppLayout'
import AuthPage from '@/pages/AuthPage'
import DashboardPage from '@/pages/DashboardPage'
import RecordsPage from '@/pages/RecordsPage'
import SourcesPage from '@/pages/SourcesPage'
import ChatPage from '@/pages/ChatPage'
import ConfigPage from '@/pages/ConfigPage'
import DomainsPage from '@/pages/DomainsPage'
import MembersPage from '@/pages/MembersPage'
import InvitationsPage from '@/pages/InvitationsPage'
import AuditLogsPage from '@/pages/AuditLogsPage'
import OAuthGoogleCallbackPage from '@/pages/OAuthGoogleCallbackPage'
import { useAuth } from '@/hooks/useAuth'

function LandingPage() {
  const { isAuthenticated } = useAuth()
  if (isAuthenticated) return <Navigate to="/dashboard" replace />

  return (
    <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)' }}>
      <div style={{ textAlign: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px', marginBottom: '16px' }}>
          <svg width="28" height="28" viewBox="0 0 22 22" fill="none">
            <rect x="1" y="1" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
            <rect x="12" y="1" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
            <rect x="1" y="12" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
            <rect x="12" y="12" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.3" />
          </svg>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '30px', lineHeight: 1, color: 'var(--text)', margin: 0, letterSpacing: '-0.02em' }}>
            Cube AI
          </h1>
        </div>
        <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '15px', color: 'var(--text-muted)', margin: '0 0 28px' }}>
          Intelligent document platform
        </p>
        <a
          href="/auth"
          style={{ background: 'var(--accent)', border: 'none', color: '#070c16', padding: '10px 24px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', fontWeight: '700', textDecoration: 'none', display: 'inline-block' }}
        >
          Get Started
        </a>
      </div>
    </div>
  )
}

function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) return (
    <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '14px' }}>
        <div style={{ width: '28px', height: '28px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>Restoring session…</span>
      </div>
    </div>
  )

  if (!isAuthenticated) {
    return <Navigate to="/auth" state={{ from: location.pathname }} replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/auth" element={<AuthPage />} />
      <Route path="/oauth/google/callback" element={<OAuthGoogleCallbackPage />} />
      <Route
        element={
          <RequireAuth>
            <AppLayout />
          </RequireAuth>
        }
      >
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/records" element={<RecordsPage />} />
        <Route path="/sources" element={<SourcesPage />} />
        <Route path="/chat" element={<ChatPage />} />
        <Route path="/config" element={<ConfigPage />} />
        <Route path="/domains" element={<DomainsPage />} />
        <Route path="/members" element={<MembersPage />} />
        <Route path="/invitations" element={<InvitationsPage />} />
        <Route path="/audit-logs" element={<AuditLogsPage />} />
      </Route>
    </Routes>
  )
}
