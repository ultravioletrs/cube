// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { ATTESTATION_ENABLED } from '@/lib/features'
import { userDisplayName, userEmail, userInitials } from '@/lib/auth/user-display'
import type { ActiveWorkspace } from '@/types'

const CubeAILogo = () => (
  <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
    <rect x="1" y="1" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
    <rect x="12" y="1" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
    <rect x="1" y="12" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.9" />
    <rect x="12" y="12" width="9" height="9" rx="1.5" fill="var(--accent)" opacity="0.3" />
  </svg>
)

interface NavItem {
  id: string
  label: string
  path: string
  icon: React.ReactNode
  external?: boolean
}

const ATOM_UI_URL = import.meta.env.VITE_ATOM_UI_URL ?? '/atom'

const overviewItems: NavItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    path: '/dashboard',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <rect x="2" y="2" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="11" y="2" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="2" y="11" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <rect x="11" y="11" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.5" opacity="0.4" />
      </svg>
    ),
  },
]

const knowledgeItems: NavItem[] = [
  {
    id: 'records',
    label: 'Records',
    path: '/records',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <rect x="3" y="2" width="10" height="13" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M6 6h4M6 9h4M6 12h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <rect x="7" y="5" width="10" height="13" rx="1.5" stroke="currentColor" strokeWidth="1.5" fill="var(--sidebar-bg)" />
        <path d="M10 9h4M10 12h4M10 15h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: 'sources',
    label: 'Sources',
    path: '/sources',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <ellipse cx="10" cy="5.5" rx="7" ry="2.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M3 5.5v9c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5v-9" stroke="currentColor" strokeWidth="1.5" />
        <path d="M3 10c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    ),
  },
]

const aiItems: NavItem[] = [
  {
    id: 'chat',
    label: 'Prompt',
    path: '/chat',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <path d="M3 4.5A1.5 1.5 0 014.5 3h11A1.5 1.5 0 0117 4.5v8A1.5 1.5 0 0115.5 14H11l-3 3v-3H4.5A1.5 1.5 0 013 12.5v-8z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
        <path d="M7 8h6M7 11h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: 'guardrails',
    label: 'Guardrails',
    path: '/guardrails',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <path d="M10 2L3 5.5v4.5c0 3.8 2.9 7.4 7 8.5 4.1-1.1 7-4.7 7-8.5V5.5L10 2z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
        <path d="M7 10l2 2 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
  },
]

const platformItems: NavItem[] = [
  {
    id: 'workspaces',
    label: 'Workspaces',
    path: '/workspaces',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="7.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M10 2.5C10 2.5 7 6 7 10s3 7.5 3 7.5M10 2.5c0 0 3 3.5 3 7.5s-3 7.5-3 7.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M2.5 10h15" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: 'members',
    label: 'Members',
    path: '/members',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <circle cx="7" cy="6" r="3" stroke="currentColor" strokeWidth="1.5" />
        <path d="M2.5 17c.6-3 2.2-4.5 4.5-4.5s3.9 1.5 4.5 4.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <circle cx="14" cy="7" r="2" stroke="currentColor" strokeWidth="1.5" opacity="0.7" />
        <path d="M12.5 13c1.8.2 3 1.5 3.5 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.7" />
      </svg>
    ),
  },
  {
    id: 'invitations',
    label: 'Invitations',
    path: '/invitations',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <rect x="3" y="5" width="14" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M4 6l6 5 6-5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M15 3v4M13 5h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  {
    id: 'audit-logs',
    label: 'Audit Logs',
    path: '/audit-logs',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <path d="M5 3h8l2 2v12H5V3z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
        <path d="M13 3v3h3M8 9h5M8 12h5M8 15h3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
  ...(ATTESTATION_ENABLED ? [{
    id: 'attestation',
    label: 'Attestation',
    path: '/attestation',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <path d="M10 2L3 5v5c0 4.418 3.134 8.547 7 9.5C13.866 18.547 17 14.418 17 10V5l-7-3z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
        <path d="M7 10l2 2 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
  }] : []),
]

// Identity & Access is the ATOM admin UI; only admins should see it.
const adminItem: NavItem = {
  id: 'advanced-identity-access',
  label: 'Advanced IAM',
  path: ATOM_UI_URL,
  external: true,
  icon: (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
      <path d="M10 2.5l6.5 3v4.8c0 3.6-2.7 6.9-6.5 7.9-3.8-1-6.5-4.3-6.5-7.9V5.5l6.5-3z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M7.5 10.2l1.7 1.7 3.4-3.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
}

const bottomItems: NavItem[] = [
  {
    id: 'config',
    label: 'Settings',
    path: '/config',
    icon: (
      <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="2.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M10 3v2M10 15v2M3 10h2M15 10h2M4.93 4.93l1.41 1.41M13.66 13.66l1.41 1.41M4.93 15.07l1.41-1.41M13.66 6.34l1.41-1.41" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    ),
  },
]

function NavGroup({ label, items, currentPath }: { label: string; items: NavItem[]; currentPath: string }) {
  return (
    <div style={{ marginBottom: '4px' }}>
      <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', fontWeight: '600', color: 'var(--text-dim)', letterSpacing: '0.1em', textTransform: 'uppercase', padding: '6px 12px 2px', margin: 0 }}>
        {label}
      </p>
      {items.map(item => {
        const active = currentPath === item.path || currentPath.startsWith(item.path + '/')
        return (
          <NavButton key={item.id} item={item} active={active} />
        )
      })}
    </div>
  )
}

function NavButton({ item, active }: { item: NavItem; active: boolean }) {
  const navigate = useNavigate()
  const [hovered, setHovered] = useState(false)
  const handleClick = () => {
    if (item.external) {
      window.open(item.path, '_blank', 'noopener,noreferrer')
      return
    }
    navigate(item.path)
  }

  return (
    <button
      onClick={handleClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '9px',
        padding: '7px 12px',
        borderRadius: '7px',
        border: 'none',
        cursor: 'pointer',
        fontFamily: 'Space Grotesk, sans-serif',
        fontWeight: '500',
        fontSize: '13px',
        transition: 'all 0.12s ease',
        textAlign: 'left',
        width: '100%',
        background: active ? 'rgba(0,212,180,0.1)' : hovered ? 'rgba(255,255,255,0.04)' : 'transparent',
        color: active ? 'var(--accent)' : hovered ? 'var(--text)' : 'var(--text-muted)',
        borderLeft: active ? '2px solid var(--accent)' : '2px solid transparent',
      }}
    >
      <span style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>{item.icon}</span>
      <span style={{ flex: 1 }}>{item.label}</span>
    </button>
  )
}

function WorkspaceBadge({ activeWorkspace, onClick }: { activeWorkspace: { name: string } | null; onClick: () => void }) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      title="Switch workspace"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '6px 12px',
        margin: '0 8px 8px',
        borderRadius: '8px',
        border: '1px solid var(--border)',
        cursor: 'pointer',
        background: hovered ? 'rgba(255,255,255,0.04)' : 'rgba(255,255,255,0.02)',
        width: 'calc(100% - 16px)',
        textAlign: 'left',
        transition: 'all 0.12s ease',
      }}
    >
      <div style={{ width: '20px', height: '20px', borderRadius: '5px', background: 'var(--accent)', opacity: 0.85, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '10px', color: '#070c16' }}>
          {activeWorkspace ? activeWorkspace.name[0].toUpperCase() : '?'}
        </span>
      </div>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {activeWorkspace ? activeWorkspace.name : 'No workspace'}
        </p>
        <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>
          {activeWorkspace ? 'workspace' : 'select workspace'}
        </p>
      </div>
    </button>
  )
}

function UserMenuInline() {
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  const initials = user ? userInitials(user) : '?'
  const name = user ? userDisplayName(user) : 'User'
  const email = user ? userEmail(user) : ''

  return (
    <div style={{ position: 'relative', padding: '8px 12px', borderTop: '1px solid var(--border)' }}>
      <button
        onClick={() => setOpen(o => !o)}
        style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}
      >
        <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: 'rgba(0,212,180,0.15)', border: '1px solid rgba(0,212,180,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
          <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '11px', color: 'var(--accent)' }}>{initials}</span>
        </div>
        <div style={{ flex: 1, textAlign: 'left', overflow: 'hidden' }}>
          <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {name}
          </p>
          <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {email}
          </p>
        </div>
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" style={{ color: 'var(--text-dim)', flexShrink: 0 }}>
          <path d="M2 4l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>

      {open && (
        <div style={{ position: 'absolute', bottom: '100%', left: '8px', right: '8px', background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '10px', padding: '4px', boxShadow: '0 -8px 32px rgba(0,0,0,0.4)', zIndex: 50 }}>
          <button
            onClick={() => { setOpen(false); navigate('/config') }}
            style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', padding: '7px 10px', background: 'none', border: 'none', cursor: 'pointer', borderRadius: '7px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', textAlign: 'left' }}
            onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,255,255,0.05)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}
          >
            <svg width="14" height="14" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="10" r="2.5" stroke="currentColor" strokeWidth="1.5" />
              <path d="M10 3v2M10 15v2M3 10h2M15 10h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
            Settings
          </button>
          <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />
          <button
            onClick={() => { setOpen(false); logout() }}
            style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', padding: '7px 10px', background: 'none', border: 'none', cursor: 'pointer', borderRadius: '7px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b', textAlign: 'left' }}
            onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,107,107,0.08)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}
          >
            <svg width="14" height="14" viewBox="0 0 20 20" fill="none">
              <path d="M13 3h4a1 1 0 011 1v12a1 1 0 01-1 1h-4M9 14l4-4-4-4M3 10h10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}

export default function Sidebar({ activeWorkspace }: { activeWorkspace: ActiveWorkspace | null }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { user } = useAuth()

  const platform = user?.role === 'admin' ? [...platformItems, adminItem] : platformItems

  return (
    <aside style={{ width: '210px', minWidth: '210px', height: '100%', background: 'var(--sidebar-bg)', borderRight: '1px solid var(--border)', display: 'flex', flexDirection: 'column', position: 'relative', zIndex: 10 }}>
      {/* Logo */}
      <button
        onClick={() => navigate('/records')}
        style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '18px 16px 14px', borderBottom: '1px solid var(--border)', marginBottom: '8px', background: 'none', border: 'none', cursor: 'pointer', width: '100%', textAlign: 'left' }}
      >
        <CubeAILogo />
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '16px', lineHeight: 1, color: 'var(--text)', letterSpacing: '-0.02em' }}>
          Cube AI
        </span>
      </button>

      {/* Workspace switcher */}
      <WorkspaceBadge
        activeWorkspace={activeWorkspace}
        onClick={() => navigate('/workspaces')}
      />

      {/* Navigation */}
      <nav style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0', padding: '0 8px', overflowY: 'auto' }}>
        <NavGroup label="AI" items={aiItems} currentPath={location.pathname} />
        <NavGroup label="Overview" items={overviewItems} currentPath={location.pathname} />
        <NavGroup label="Knowledge Base" items={knowledgeItems} currentPath={location.pathname} />
        <NavGroup label="Platform" items={platform} currentPath={location.pathname} />
        <div style={{ flex: 1 }} />
        <NavGroup label="Account" items={bottomItems} currentPath={location.pathname} />
      </nav>

      {/* User */}
      <UserMenuInline />
    </aside>
  )
}
