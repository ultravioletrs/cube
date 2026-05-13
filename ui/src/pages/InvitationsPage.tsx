// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import {
  listDomainInvitations,
  listUserInvitations,
  deleteInvitation,
  acceptInvitation,
  rejectInvitation,
} from '@/lib/platform/service'
import type { Invitation } from '@/lib/platform/service'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

type Tab = 'domain' | 'mine'

function statusLabel(inv: Invitation): { label: string; color: string } {
  if (inv.rejected_at) return { label: 'rejected', color: '#ff5050' }
  if (inv.confirmed_at) return { label: 'accepted', color: '#00d4b4' }
  return { label: 'pending', color: '#ffb400' }
}

function resolveID(field: string | { id?: string } | undefined): string {
  if (!field) return '?'
  if (typeof field === 'string') return field
  return field.id ?? '?'
}

function resolveName(field: string | { email?: string; first_name?: string; last_name?: string; name?: string } | undefined): string {
  if (!field) return '?'
  if (typeof field === 'string') return field.slice(0, 8) + '…'
  if (field.email) return field.email
  if (field.name) return field.name
  return '?'
}

function InvitationRow({
  inv,
  tab,
  onDelete,
  onAccept,
  onReject,
}: {
  inv: Invitation
  tab: Tab
  onDelete: () => void
  onAccept?: () => void
  onReject?: () => void
}) {
  const { label, color } = statusLabel(inv)
  const isPending = !inv.confirmed_at && !inv.rejected_at
  const domainID = resolveID(inv.domain_id as string | { id?: string })
  const inviteeName = resolveName(inv.invitee_user_id as string | { email?: string })

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <p style={{ margin: '0 0 2px', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {tab === 'domain' ? inviteeName : (inv.domain_name ?? domainID.slice(0, 8) + '…')}
        </p>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {inv.role_name && (
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
              {inv.role_name}
            </span>
          )}
          {inv.created_at && (
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
              {new Date(inv.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
            </span>
          )}
        </div>
      </div>

      <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px', padding: '3px 8px', borderRadius: '20px', background: color + '18', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', fontWeight: '500', color, flexShrink: 0 }}>
        <span style={{ width: '4px', height: '4px', borderRadius: '50%', background: color }} />
        {label}
      </span>

      {isPending && tab === 'mine' && (
        <div style={{ display: 'flex', gap: '6px', flexShrink: 0 }}>
          <button
            onClick={onAccept}
            style={{ padding: '4px 10px', borderRadius: '6px', border: '1px solid rgba(0,212,180,0.3)', background: 'rgba(0,212,180,0.08)', color: 'var(--accent)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', fontWeight: '600', cursor: 'pointer' }}
          >
            Accept
          </button>
          <button
            onClick={onReject}
            style={{ padding: '4px 10px', borderRadius: '6px', border: '1px solid rgba(255,80,80,0.2)', background: 'rgba(255,80,80,0.06)', color: '#ff5050', fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', fontWeight: '600', cursor: 'pointer' }}
          >
            Reject
          </button>
        </div>
      )}

      {tab === 'domain' && (
        <button
          onClick={onDelete}
          title="Delete invitation"
          style={{ padding: '5px', borderRadius: '6px', border: 'none', background: 'none', cursor: 'pointer', color: 'var(--text-dim)', display: 'flex', alignItems: 'center', flexShrink: 0 }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none">
            <path d="M2 4h12M5 4V2h6v2M6 7v5M10 7v5M3 4l1 9a1 1 0 001 1h6a1 1 0 001-1l1-9" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      )}
    </div>
  )
}

export default function InvitationsPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [tab, setTab] = useState<Tab>('domain')
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!tokens?.accessToken) return
    if (tab === 'domain' && !activeDomain?.id) return
    setLoading(true)
    setError(null)
    try {
      const res = tab === 'domain'
        ? await listDomainInvitations(activeDomain!.id, tokens.accessToken)
        : await listUserInvitations(tokens.accessToken)
      setInvitations(res.invitations)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load invitations')
    } finally {
      setLoading(false)
    }
  }, [tokens?.accessToken, activeDomain?.id, tab])

  useEffect(() => { void load() }, [load])

  async function handleDelete(inv: Invitation) {
    if (!tokens?.accessToken) return
    const userID = resolveID(inv.invitee_user_id as string | { id?: string })
    const domainID = resolveID(inv.domain_id as string | { id?: string })
    try {
      await deleteInvitation(userID, domainID, tokens.accessToken)
      setInvitations(prev => prev.filter(i => i !== inv))
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete invitation')
    }
  }

  async function handleAccept(inv: Invitation) {
    if (!tokens?.accessToken) return
    const domainID = resolveID(inv.domain_id as string | { id?: string })
    try {
      await acceptInvitation(domainID, tokens.accessToken)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to accept invitation')
    }
  }

  async function handleReject(inv: Invitation) {
    if (!tokens?.accessToken) return
    const domainID = resolveID(inv.domain_id as string | { id?: string })
    try {
      await rejectInvitation(domainID, tokens.accessToken)
      void load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to reject invitation')
    }
  }

  const tabStyle = (t: Tab): React.CSSProperties => ({
    padding: '6px 14px',
    borderRadius: '7px',
    border: 'none',
    cursor: 'pointer',
    fontFamily: 'Space Grotesk, sans-serif',
    fontSize: '13px',
    fontWeight: '600',
    background: tab === t ? 'rgba(0,212,180,0.1)' : 'transparent',
    color: tab === t ? 'var(--accent)' : 'var(--text-muted)',
    transition: 'all 0.12s ease',
  })

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>Invitations</h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            Manage domain invitations
          </p>
        </div>
        <UserMenu />
      </div>

      {/* Tabs */}
      <div style={{ display: 'flex', gap: '4px', padding: '12px 28px 0', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <button style={tabStyle('domain')} onClick={() => setTab('domain')}>Domain</button>
        <button style={tabStyle('mine')} onClick={() => setTab('mine')}>My invitations</button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {tab === 'domain' && !activeDomain && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '200px' }}>
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>
              Select a domain first to view its invitations.
            </p>
          </div>
        )}

        {loading && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '200px' }}>
            <div style={{ width: '24px', height: '24px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          </div>
        )}

        {!loading && error && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {error}
          </div>
        )}

        {!loading && !error && invitations.length === 0 && (tab === 'mine' || activeDomain) && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '200px' }}>
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>
              No invitations
            </p>
          </div>
        )}

        {!loading && !error && invitations.length > 0 && (
          <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', overflow: 'hidden' }}>
            <div style={{ display: 'flex', alignItems: 'center', padding: '10px 16px', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)' }}>
              <span style={{ flex: 1, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', fontWeight: '600', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
                {tab === 'domain' ? 'Invitee' : 'Domain'}
              </span>
              <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', fontWeight: '600', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Status</span>
            </div>
            {invitations.map((inv, i) => (
              <InvitationRow
                key={i}
                inv={inv}
                tab={tab}
                onDelete={() => handleDelete(inv)}
                onAccept={() => handleAccept(inv)}
                onReject={() => handleReject(inv)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
