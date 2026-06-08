// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import UserMenu from '@/components/UserMenu'
import { listIdentityAuditLogs } from '@/lib/platform/service'
import type { IdentityAuditLog } from '@/lib/platform/service'
import type { AppContext } from '@/types'

function outcomeColor(outcome: string): string {
  switch (outcome.toLowerCase()) {
    case 'allow':
    case 'success':
      return 'var(--accent)'
    case 'deny':
    case 'error':
    case 'fail':
      return '#ff6b6b'
    default:
      return '#ffb400'
  }
}

function formatDetails(details: Record<string, unknown>): string {
  const text = JSON.stringify(details ?? {})
  return text.length > 180 ? `${text.slice(0, 177)}...` : text
}

export default function AuditLogsPage() {
  const { activeWorkspace } = useOutletContext<AppContext>()
  const [logs, setLogs] = useState<IdentityAuditLog[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setLogs(await listIdentityAuditLogs(activeWorkspace?.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit logs')
    } finally {
      setLoading(false)
    }
  }, [activeWorkspace?.id])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void load() }, [load])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px' }}>
            Audit Logs
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            ATOM identity and authorization activity{activeWorkspace?.name ? ` for ${activeWorkspace.name}` : ''}
          </p>
        </div>
        <UserMenu />
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {loading && <div style={{ color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>Loading audit logs...</div>}
        {error && <div style={{ padding: '12px 14px', marginBottom: '14px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '8px', color: '#ff6b6b', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>{error}</div>}

        {!loading && !error && (
          <div style={{ border: '1px solid var(--border)', borderRadius: '10px', overflow: 'hidden', background: 'var(--card-bg)' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr 110px 1.2fr', gap: '12px', padding: '10px 14px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textTransform: 'uppercase' }}>
              <span>Time</span>
              <span>Event</span>
              <span>Outcome</span>
              <span>Details</span>
            </div>
            {logs.length === 0 ? (
              <div style={{ padding: '28px 14px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>No identity audit logs found.</div>
            ) : logs.map(log => (
              <div key={log.id} style={{ display: 'grid', gridTemplateColumns: '150px 1fr 110px 1.2fr', gap: '12px', alignItems: 'center', padding: '12px 14px', borderBottom: '1px solid var(--border)' }}>
                <span style={{ color: 'var(--text-muted)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}>{new Date(log.createdAt).toLocaleString()}</span>
                <div style={{ minWidth: 0 }}>
                  <p style={{ margin: 0, color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{log.event}</p>
                  <p style={{ margin: '2px 0 0', color: 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{log.entityId ?? 'system'}</p>
                </div>
                <span style={{ color: outcomeColor(log.outcome), fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>{log.outcome}</span>
                <code style={{ color: 'var(--text-muted)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{formatDetails(log.details)}</code>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
