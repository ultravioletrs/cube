// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import UserMenu from '@/components/UserMenu'
import {
  acceptWorkspaceInvitation,
  createWorkspaceInvitation,
  listMyWorkspaceInvitations,
  listWorkspaceInvitations,
  rejectWorkspaceInvitation,
  revokeWorkspaceInvitation,
} from '@/lib/platform/service'
import type { WorkspaceInvitation } from '@/lib/platform/service'
import type { AppContext } from '@/types'

function invitationStatus(invitation: WorkspaceInvitation): string {
  if (invitation.revokedAt) return 'revoked'
  if (invitation.acceptedAt) return 'accepted'
  if (invitation.rejectedAt) return 'rejected'
  return 'pending'
}

function statusColor(status: string): string {
  switch (status) {
    case 'pending':
      return '#ffb400'
    case 'accepted':
      return 'var(--accent)'
    case 'revoked':
    case 'rejected':
      return '#ff6b6b'
    default:
      return 'var(--text-muted)'
  }
}

function CreateInvitationModal({
  onClose,
  onCreate,
}: {
  onClose: () => void
  onCreate: (email: string) => Promise<void>
}) {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!email.trim()) return
    setLoading(true)
    setError(null)
    try {
      await onCreate(email.trim())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create invitation')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }} onClick={onClose}>
      <form onSubmit={handleSubmit} style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '24px', width: '360px', boxShadow: '0 24px 80px rgba(0,0,0,0.5)' }} onClick={e => e.stopPropagation()}>
        <h2 style={{ margin: '0 0 4px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '18px', fontWeight: 700 }}>Invite member</h2>
        <p style={{ margin: '0 0 18px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>Send a workspace invitation through ATOM.</p>
        <label style={{ display: 'block', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, marginBottom: '6px' }} htmlFor="invite-email">Email</label>
        <input
          id="invite-email"
          type="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          autoFocus
          style={{ width: '100%', boxSizing: 'border-box', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', outline: 'none' }}
        />
        {error && <p style={{ margin: '12px 0 0', color: '#ff6b6b', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px' }}>{error}</p>}
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end', marginTop: '20px' }}>
          <button type="button" onClick={onClose} style={{ padding: '8px 14px', border: '1px solid var(--border)', borderRadius: '8px', background: 'transparent', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>Cancel</button>
          <button type="submit" disabled={loading || !email.trim()} style={{ padding: '8px 16px', border: 'none', borderRadius: '8px', background: loading || !email.trim() ? 'rgba(0,212,180,0.4)' : 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: 700, cursor: loading || !email.trim() ? 'not-allowed' : 'pointer' }}>{loading ? 'Sending...' : 'Send'}</button>
        </div>
      </form>
    </div>
  )
}

export default function InvitationsPage() {
  const { activeWorkspace } = useOutletContext<AppContext>()
  const [invitations, setInvitations] = useState<WorkspaceInvitation[]>([])
  const [myInvitations, setMyInvitations] = useState<WorkspaceInvitation[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [workspaceItems, myItems] = await Promise.all([
        activeWorkspace?.id ? listWorkspaceInvitations(activeWorkspace.id) : Promise.resolve([]),
        listMyWorkspaceInvitations(),
      ])
      setInvitations(workspaceItems)
      setMyInvitations(myItems)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load invitations')
    } finally {
      setLoading(false)
    }
  }, [activeWorkspace?.id])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void load() }, [load])

  async function handleCreate(email: string) {
    if (!activeWorkspace?.id) return
    const created = await createWorkspaceInvitation(activeWorkspace.id, email)
    setInvitations(prev => [created, ...prev])
    setShowCreate(false)
  }

  async function handleRevoke(invitation: WorkspaceInvitation) {
    if (!activeWorkspace?.id) return
    await revokeWorkspaceInvitation(activeWorkspace.id, invitation.id)
    await load()
  }

  async function handleAccept(invitation: WorkspaceInvitation) {
    await acceptWorkspaceInvitation(invitation.tenantId)
    await load()
  }

  async function handleReject(invitation: WorkspaceInvitation) {
    await rejectWorkspaceInvitation(invitation.tenantId)
    await load()
  }

  const rowStyle: React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns: '1.6fr 0.8fr 0.8fr 120px',
    gap: '12px',
    alignItems: 'center',
    padding: '12px 14px',
    borderBottom: '1px solid var(--border)',
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px' }}>
            Invitations
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            Invite users and respond to workspace access requests
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button
            type="button"
            onClick={() => setShowCreate(true)}
            disabled={!activeWorkspace?.id}
            style={{ padding: '8px 14px', background: activeWorkspace?.id ? 'var(--accent)' : 'rgba(0,212,180,0.35)', border: 'none', borderRadius: '8px', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: 700, cursor: activeWorkspace?.id ? 'pointer' : 'not-allowed' }}
          >
            Invite member
          </button>
          <UserMenu />
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px', display: 'flex', flexDirection: 'column', gap: '18px' }}>
        {loading && <div style={{ color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>Loading invitations...</div>}
        {error && <div style={{ padding: '12px 14px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '8px', color: '#ff6b6b', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>{error}</div>}

        <section>
          <h2 style={{ margin: '0 0 10px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '15px', fontWeight: 700 }}>Workspace invitations</h2>
          <div style={{ border: '1px solid var(--border)', borderRadius: '10px', overflow: 'hidden', background: 'var(--card-bg)' }}>
            <div style={{ ...rowStyle, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textTransform: 'uppercase' }}>
              <span>Invitee</span><span>Role</span><span>Status</span><span />
            </div>
            {invitations.length === 0 ? (
              <div style={{ padding: '22px 14px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
                {activeWorkspace?.id ? 'No workspace invitations found.' : 'Select a workspace to view invitations.'}
              </div>
            ) : invitations.map(invitation => {
              const status = invitationStatus(invitation)
              return (
                <div key={invitation.id} style={rowStyle}>
                  <span style={{ color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{invitation.inviteeEmail ?? invitation.inviteeUserId ?? invitation.id}</span>
                  <span style={{ color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>{invitation.roleName ?? 'member'}</span>
                  <span style={{ color: statusColor(status), fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>{status}</span>
                  <button type="button" disabled={status !== 'pending'} onClick={() => void handleRevoke(invitation)} style={{ justifySelf: 'end', padding: '5px 10px', border: '1px solid rgba(255,80,80,0.3)', borderRadius: '6px', background: 'transparent', color: status === 'pending' ? '#ff5050' : 'var(--text-dim)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, cursor: status === 'pending' ? 'pointer' : 'not-allowed' }}>Revoke</button>
                </div>
              )
            })}
          </div>
        </section>

        <section>
          <h2 style={{ margin: '0 0 10px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '15px', fontWeight: 700 }}>My invitations</h2>
          <div style={{ border: '1px solid var(--border)', borderRadius: '10px', overflow: 'hidden', background: 'var(--card-bg)' }}>
            {myInvitations.length === 0 ? (
              <div style={{ padding: '22px 14px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>No invitations waiting for you.</div>
            ) : myInvitations.map(invitation => (
              <div key={invitation.id} style={{ ...rowStyle, gridTemplateColumns: '1.6fr 0.8fr 160px' }}>
                <span style={{ color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>{invitation.tenantId}</span>
                <span style={{ color: statusColor(invitationStatus(invitation)), fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>{invitationStatus(invitation)}</span>
                <span style={{ justifySelf: 'end', display: 'flex', gap: '8px' }}>
                  <button type="button" onClick={() => void handleAccept(invitation)} style={{ padding: '5px 10px', border: 'none', borderRadius: '6px', background: 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 700, cursor: 'pointer' }}>Accept</button>
                  <button type="button" onClick={() => void handleReject(invitation)} style={{ padding: '5px 10px', border: '1px solid var(--border)', borderRadius: '6px', background: 'transparent', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}>Reject</button>
                </span>
              </div>
            ))}
          </div>
        </section>
      </div>

      {showCreate && <CreateInvitationModal onClose={() => setShowCreate(false)} onCreate={handleCreate} />}
    </div>
  )
}
