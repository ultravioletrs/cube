// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import UserMenu from '@/components/UserMenu'
import { listWorkspaceMembers, removeWorkspaceMember } from '@/lib/platform/service'
import type { WorkspaceMember } from '@/lib/platform/service'
import type { AppContext } from '@/types'

function statusColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'enabled':
    case 'active':
      return 'var(--accent)'
    case 'disabled':
    case 'inactive':
      return '#ffb400'
    default:
      return 'var(--text-muted)'
  }
}

export default function MembersPage() {
  const { activeWorkspace } = useOutletContext<AppContext>()
  const [members, setMembers] = useState<WorkspaceMember[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const load = useCallback(async () => {
    if (!activeWorkspace?.id) return
    setLoading(true)
    setError(null)
    try {
      setMembers(await listWorkspaceMembers(activeWorkspace.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load members')
    } finally {
      setLoading(false)
    }
  }, [activeWorkspace?.id])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void load() }, [load])

  async function handleRemove(member: WorkspaceMember) {
    if (!activeWorkspace?.id) return
    if (!window.confirm(`Remove "${member.name}" from this workspace?`)) return
    setActionError(null)
    try {
      await removeWorkspaceMember(activeWorkspace.id, member.id)
      setMembers(prev => prev.filter(item => item.id !== member.id))
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to remove member')
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px' }}>
            Members
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            People and entities with access to {activeWorkspace?.name ?? 'the selected workspace'}
          </p>
        </div>
        <UserMenu />
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {!activeWorkspace?.id && (
          <div style={{ padding: '16px', border: '1px solid var(--border)', borderRadius: '10px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
            Select a workspace to view members.
          </div>
        )}

        {loading && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '180px' }}>
            <div style={{ width: '24px', height: '24px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          </div>
        )}

        {(error || actionError) && (
          <div style={{ padding: '12px 14px', marginBottom: '14px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '8px', color: '#ff6b6b', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
            {error ?? actionError}
          </div>
        )}

        {!loading && activeWorkspace?.id && !error && (
          <div style={{ border: '1px solid var(--border)', borderRadius: '10px', overflow: 'hidden', background: 'var(--card-bg)' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1.7fr 1fr 1fr 90px', gap: '12px', padding: '10px 14px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textTransform: 'uppercase' }}>
              <span>Name</span>
              <span>Kind</span>
              <span>Status</span>
              <span />
            </div>
            {members.length === 0 ? (
              <div style={{ padding: '28px 14px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
                No members found.
              </div>
            ) : members.map(member => (
              <div key={member.id} style={{ display: 'grid', gridTemplateColumns: '1.7fr 1fr 1fr 90px', gap: '12px', alignItems: 'center', padding: '12px 14px', borderBottom: '1px solid var(--border)' }}>
                <div style={{ minWidth: 0 }}>
                  <p style={{ margin: 0, color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {member.name}
                  </p>
                  <p style={{ margin: '2px 0 0', color: 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {member.id}
                  </p>
                </div>
                <span style={{ color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>{member.kind}</span>
                <span style={{ color: statusColor(member.status), fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>{member.status}</span>
                <button
                  type="button"
                  onClick={() => void handleRemove(member)}
                  style={{ justifySelf: 'end', padding: '5px 10px', border: '1px solid rgba(255,80,80,0.3)', borderRadius: '6px', background: 'transparent', color: '#ff5050', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
