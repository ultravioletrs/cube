// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { Navigate, Routes, Route } from 'react-router-dom'
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
import GuardrailsPage from '@/pages/GuardrailsPage'
import OAuthGoogleCallbackPage from '@/pages/OAuthGoogleCallbackPage'
import AttestationPage from '@/pages/AttestationPage'
import { useAuth } from '@/hooks/useAuth'
import { ATTESTATION_ENABLED } from '@/lib/features'

function RootRedirect() {
  const { isAuthenticated } = useAuth()
  if (isAuthenticated) return <Navigate to="/domains" replace />
  return <Navigate to="/login" replace />
}

function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) return (
    <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '14px' }}>
        <div style={{ width: '28px', height: '28px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>Restoring session…</span>
      </div>
    </div>
  )

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: '/domains' }} replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<RootRedirect />} />
      <Route path="/login" element={<AuthPage />} />
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
        <Route path="/prompt" element={<ChatPage />} />
        <Route path="/config" element={<ConfigPage />} />
        <Route path="/domains" element={<DomainsPage />} />
        <Route path="/members" element={<MembersPage />} />
        <Route path="/invitations" element={<InvitationsPage />} />
        <Route path="/audit-logs" element={<AuditLogsPage />} />
        {ATTESTATION_ENABLED && (
          <Route path="/attestation" element={<AttestationPage />} />
        )}
        <Route path="/guardrails" element={<GuardrailsPage />} />
      </Route>
    </Routes>
  )
}
