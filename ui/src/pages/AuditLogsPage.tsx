import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { listUserJournals, listEntityJournals, JOURNAL_PAGE_SIZE } from '@/lib/platform/service'
import type { Journal } from '@/lib/platform/service'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

type Tab = 'user' | 'domain'

function formatDate(s?: string) {
  if (!s) return '—'
  try {
    return new Date(s).toLocaleString(undefined, {
      dateStyle: 'medium',
      timeStyle: 'medium',
    })
  } catch {
    return s
  }
}

function OperationBadge({ op }: { op?: string }) {
  if (!op) return null
  const lower = op.toLowerCase()
  let bg = 'rgba(255,255,255,0.06)'
  let color = 'var(--text-muted)'
  if (lower.includes('create') || lower.includes('add') || lower.includes('register')) {
    bg = 'rgba(0,212,180,0.1)'; color = '#00d4b4'
  } else if (lower.includes('delete') || lower.includes('remove') || lower.includes('revoke')) {
    bg = 'rgba(255,80,80,0.1)'; color = '#ff5050'
  } else if (lower.includes('update') || lower.includes('assign') || lower.includes('enable') || lower.includes('disable')) {
    bg = 'rgba(255,180,0,0.1)'; color = '#ffb400'
  } else if (lower.includes('login') || lower.includes('issue') || lower.includes('refresh')) {
    bg = 'rgba(100,160,255,0.1)'; color = '#64a0ff'
  }
  return (
    <span style={{ display: 'inline-block', background: bg, color, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '2px 8px', borderRadius: '20px', fontWeight: '500', whiteSpace: 'nowrap' }}>
      {op}
    </span>
  )
}

function PayloadCell({ payload }: { payload?: Record<string, unknown> }) {
  const [open, setOpen] = useState(false)
  if (!payload || Object.keys(payload).length === 0) return <span style={{ color: 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>—</span>
  return (
    <div>
      <button
        onClick={() => setOpen(o => !o)}
        style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border)', borderRadius: '5px', color: 'var(--text-muted)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '2px 8px', cursor: 'pointer' }}
      >
        {open ? 'hide' : 'view'}
      </button>
      {open && (
        <pre style={{ marginTop: '6px', background: 'rgba(0,0,0,0.3)', border: '1px solid var(--border)', borderRadius: '6px', padding: '8px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-muted)', overflowX: 'auto', whiteSpace: 'pre-wrap', maxWidth: '320px' }}>
          {JSON.stringify(payload, null, 2)}
        </pre>
      )}
    </div>
  )
}

export default function AuditLogsPage() {
  const { tokens, user } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [tab, setTab] = useState<Tab>('user')
  const [journals, setJournals] = useState<Journal[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(0)
  const [total, setTotal] = useState(0)

  const load = useCallback(async () => {
    if (!tokens?.accessToken) return
    setLoading(true)
    setError(null)
    try {
      if (tab === 'user') {
        const userID = user?.id ?? ''
        if (!userID) throw new Error('User ID not available')
        const res = await listUserJournals(userID, tokens.accessToken, page)
        setJournals(res.journals)
        setTotal(res.total)
      } else {
        if (!activeDomain?.id) throw new Error('No domain selected')
        const res = await listEntityJournals('domain', activeDomain.id, activeDomain.id, tokens.accessToken, page)
        setJournals(res.journals)
        setTotal(res.total)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit logs')
    } finally {
      setLoading(false)
    }
  }, [tokens?.accessToken, tab, page, activeDomain?.id, user])

  useEffect(() => { void load() }, [load])

  function switchTab(t: Tab) {
    setTab(t)
    setPage(0)
    setJournals([])
  }

  const totalPages = Math.max(1, Math.ceil(total / JOURNAL_PAGE_SIZE))

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>
            Audit Logs
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            Activity history for your account and domains
          </p>
        </div>
        <UserMenu />
      </div>

      {/* Tabs */}
      <div style={{ display: 'flex', gap: '4px', padding: '12px 28px 0', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        {(['user', 'domain'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => switchTab(t)}
            style={{
              padding: '7px 16px',
              borderRadius: '8px 8px 0 0',
              border: 'none',
              background: tab === t ? 'var(--card-bg)' : 'transparent',
              color: tab === t ? 'var(--text)' : 'var(--text-muted)',
              fontFamily: 'Space Grotesk, sans-serif',
              fontSize: '13px',
              fontWeight: tab === t ? '700' : '500',
              cursor: 'pointer',
              borderBottom: tab === t ? '2px solid var(--accent)' : '2px solid transparent',
            }}
          >
            {t === 'user' ? 'My activity' : 'Domain activity'}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '16px 28px' }}>
        {tab === 'domain' && !activeDomain && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,180,0,0.06)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ffb400' }}>
            Select a domain from the Domains page to view domain activity.
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

        {!loading && !error && journals.length === 0 && (tab === 'user' || activeDomain) && (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '200px', gap: '10px' }}>
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>No events found</p>
          </div>
        )}

        {!loading && !error && journals.length > 0 && (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  {['Time', 'Operation', 'Payload'].map(h => (
                    <th key={h} style={{ padding: '8px 12px', textAlign: 'left', fontWeight: '600', fontSize: '11px', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.05em', whiteSpace: 'nowrap' }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {journals.map((j, i) => (
                  <tr
                    key={j.id ?? i}
                    style={{ borderBottom: '1px solid rgba(255,255,255,0.04)' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.02)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={{ padding: '10px 12px', color: 'var(--text-muted)', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', whiteSpace: 'nowrap' }}>
                      {formatDate(j.occurred_at)}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <OperationBadge op={j.operation} />
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <PayloadCell payload={j.payload} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {!loading && !error && totalPages > 1 && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '8px', marginTop: '16px' }}>
            <button
              onClick={() => setPage(p => Math.max(0, p - 1))}
              disabled={page === 0}
              style={{ padding: '6px 14px', borderRadius: '7px', border: '1px solid var(--border)', background: 'transparent', color: page === 0 ? 'var(--text-dim)' : 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '600', cursor: page === 0 ? 'not-allowed' : 'pointer' }}
            >
              Previous
            </button>
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>
              {page + 1} / {totalPages}
            </span>
            <button
              onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
              style={{ padding: '6px 14px', borderRadius: '7px', border: '1px solid var(--border)', background: 'transparent', color: page >= totalPages - 1 ? 'var(--text-dim)' : 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '600', cursor: page >= totalPages - 1 ? 'not-allowed' : 'pointer' }}
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
