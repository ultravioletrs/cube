import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { deleteRecord, listRecords, retryRecordIngest, uploadRecordFile } from '@/lib/embedder/service'
import type { AppContext, AppRecord } from '@/types'
import UserMenu from '@/components/UserMenu'
import AddRecordModal from '@/components/AddRecordModal'

const statusColors = {
  indexed:    { bg: 'rgba(0,212,180,0.1)',  color: '#00d4b4', dot: '#00d4b4' },
  processing: { bg: 'rgba(255,180,0,0.1)',  color: '#ffb400', dot: '#ffb400' },
  error:      { bg: 'rgba(255,80,80,0.1)',  color: '#ff5050', dot: '#ff5050' },
}

function StatusBadge({ status }: { status: AppRecord['status'] }) {
  const c = statusColors[status] ?? statusColors.indexed
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', background: c.bg, color: c.color, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '3px 8px', borderRadius: '20px', fontWeight: '500' }}>
      <span style={{ width: '5px', height: '5px', borderRadius: '50%', background: c.dot, ...(status === 'processing' ? { animation: 'pulse 1.5s ease-in-out infinite' } : {}) }} />
      {status}
    </span>
  )
}

function DocBadge({ format }: { format: string }) {
  const colors: Record<string, string> = { pdf: '#ff6b6b', docx: '#4d9ef7', md: '#a78bfa' }
  const color = colors[format] ?? '#9ca3af'
  return (
    <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: color + '18', border: `1px solid ${color}30`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color, fontWeight: '700', letterSpacing: '0.05em', flexShrink: 0 }}>
      {format.toUpperCase()}
    </div>
  )
}

function ImageRecordIcon() {
  return (
    <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: 'rgba(167,139,250,0.12)', border: '1px solid rgba(167,139,250,0.25)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
      <svg width="17" height="17" viewBox="0 0 20 20" fill="none">
        <rect x="2" y="3" width="16" height="14" rx="2" stroke="#a78bfa" strokeWidth="1.4"/>
        <circle cx="7" cy="8" r="1.5" fill="#a78bfa" opacity="0.7"/>
        <path d="M2 14l4-4 3 3 3-4 4 5" stroke="#a78bfa" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
      </svg>
    </div>
  )
}

function LinkRecordIcon() {
  return (
    <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: 'rgba(0,212,180,0.1)', border: '1px solid rgba(0,212,180,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
      <svg width="17" height="17" viewBox="0 0 20 20" fill="none">
        <path d="M8.5 11.5a4.5 4.5 0 006.364 0l2-2a4.5 4.5 0 00-6.364-6.364l-1.5 1.5" stroke="#00d4b4" strokeWidth="1.4" strokeLinecap="round"/>
        <path d="M11.5 8.5a4.5 4.5 0 00-6.364 0l-2 2a4.5 4.5 0 006.364 6.364l1.5-1.5" stroke="#00d4b4" strokeWidth="1.4" strokeLinecap="round"/>
      </svg>
    </div>
  )
}

function RecordIcon({ record }: { record: AppRecord }) {
  if (record.format === 'image') return <ImageRecordIcon />
  if (record.format === 'link') return <LinkRecordIcon />
  return <DocBadge format={record.format} />
}

function recordSubtext(record: AppRecord): string {
  if (record.format === 'link') return record.url ?? ''
  if (record.format === 'image') {
    return record.chunks != null ? `${record.chunks} chunks` : 'indexing…'
  }
  if (record.chunks != null) {
    return record.pages != null ? `${record.chunks} chunks · ${record.pages} pages` : `${record.chunks} chunks`
  }
  return record.pages != null ? `${record.pages} pages · indexing…` : 'indexing…'
}

function recordDetail(record: AppRecord): string {
  if (record.format === 'link') {
    try { return new URL(record.url ?? '').hostname } catch { return '—' }
  }
  return record.size ?? '—'
}

function ErrorBanner({ message, onDismiss }: { message: string; onDismiss: () => void }) {
  return (
    <div style={{ margin: '12px 32px 0', padding: '11px 14px', background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.25)', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '10px', flexShrink: 0 }}>
      <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0 }}>
        <circle cx="7" cy="7" r="6" stroke="#ff5050" strokeWidth="1.2"/>
        <path d="M7 4v3M7 9.5v.5" stroke="#ff5050" strokeWidth="1.4" strokeLinecap="round"/>
      </svg>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12.5px', color: '#ff6b6b', flex: 1 }}>{message}</span>
      <button onClick={onDismiss} style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#ff6b6b', padding: '2px', display: 'flex', flexShrink: 0, opacity: 0.7 }}>
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/></svg>
      </button>
    </div>
  )
}

function friendlyError(msg: string): string {
  const lower = msg.toLowerCase()
  if (lower.includes('context deadline exceeded') || lower.includes('timeout'))
    return 'Indexing timed out — the document may be too large or the embedding service is overloaded.'
  if (lower.includes('connection refused') || lower.includes('no such host') || lower.includes('dial tcp'))
    return 'Could not reach the embedding service. Make sure all services are running.'
  if (lower.includes('zero chunks') || lower.includes('produced zero chunks'))
    return 'No text could be extracted from this file. Make sure it is not empty or image-only.'
  if (lower.includes('unsupported') || lower.includes('unknown format'))
    return 'This file format is not supported. Try converting it to PDF, Markdown, or plain text.'
  if (lower.includes('missing source_id') || lower.includes('missing external_id'))
    return 'The record is missing required metadata. Delete and re-upload the file.'
  if (lower.includes('pdftotext') || lower.includes('exec'))
    return 'PDF extraction failed — the embedder was updated to fix this. Click Retry Ingest.'
  return 'Indexing failed.'
}

function DetailPanel({ record, onClose, onStartChat, onRetry }: { record: AppRecord; onClose: () => void; onStartChat: (r: AppRecord) => void; onRetry: (id: string) => Promise<void> }) {
  const [retrying, setRetrying] = useState(false)
  const meta: { label: string; value: string }[] = []
  if (record.format !== 'link' && record.size) meta.push({ label: 'SIZE', value: record.size })
  meta.push({ label: 'ADDED', value: record.createdAt })
  if (record.format !== 'link' && record.format !== 'image' && record.pages != null)
    meta.push({ label: 'PAGES', value: String(record.pages) })
  meta.push({ label: 'CHUNKS', value: record.chunks != null ? `${record.chunks} vectors` : 'pending' })

  return (
    <div style={{ width: '300px', minWidth: '300px', borderLeft: '1px solid var(--border)', padding: '24px', background: 'var(--sidebar-bg)', display: 'flex', flexDirection: 'column', gap: '16px', overflow: 'auto' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
        <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px', display: 'flex', flexShrink: 0 }}>
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2 2l10 10M12 2L2 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/></svg>
        </button>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', flex: 1, minWidth: 0 }}>
          <RecordIcon record={record} />
          <div>
            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)', marginBottom: '6px', lineHeight: 1.3 }}>{record.name}</div>
            <StatusBadge status={record.status} />
          </div>
        </div>
      </div>

      {record.format === 'link' && record.url && (
        <a href={record.url} target="_blank" rel="noopener noreferrer" style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '9px 12px', background: 'rgba(0,212,180,0.05)', border: '1px solid rgba(0,212,180,0.15)', borderRadius: '8px', textDecoration: 'none', overflow: 'hidden' }}>
          <svg width="12" height="12" viewBox="0 0 20 20" fill="none" style={{ flexShrink: 0 }}>
            <path d="M8.5 11.5a4.5 4.5 0 006.364 0l2-2a4.5 4.5 0 00-6.364-6.364l-1.5 1.5" stroke="#00d4b4" strokeWidth="1.5" strokeLinecap="round"/>
            <path d="M11.5 8.5a4.5 4.5 0 00-6.364 0l-2 2a4.5 4.5 0 006.364 6.364l1.5-1.5" stroke="#00d4b4" strokeWidth="1.5" strokeLinecap="round"/>
          </svg>
          <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#00d4b4', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{record.url}</span>
        </a>
      )}

      <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12.5px', color: 'var(--text-muted)', lineHeight: 1.6 }}>{record.description}</div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
        {meta.map(({ label, value }) => (
          <div key={label} style={{ background: 'rgba(255,255,255,0.03)', borderRadius: '8px', padding: '10px 12px', border: '1px solid var(--border)' }}>
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', letterSpacing: '0.08em', marginBottom: '4px' }}>{label}</div>
            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text)', fontWeight: '600' }}>{value}</div>
          </div>
        ))}
      </div>

      {record.status === 'indexed' && (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '10px 12px', background: 'rgba(0,212,180,0.06)', borderRadius: '8px', border: '1px solid rgba(0,212,180,0.15)' }}>
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0, marginTop: '1px' }}>
            <circle cx="7" cy="7" r="6" stroke="var(--accent)" strokeWidth="1.2"/>
            <path d="M5 7l1.5 1.5L9 5" stroke="var(--accent)" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>Fully indexed · available for retrieval</span>
        </div>
      )}

      {record.status === 'error' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', padding: '10px 12px', background: 'rgba(255,80,80,0.06)', borderRadius: '8px', border: '1px solid rgba(255,80,80,0.2)' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0, marginTop: '2px' }}>
              <circle cx="7" cy="7" r="6" stroke="#ff5050" strokeWidth="1.2"/>
              <path d="M7 4v3M7 9.5v.5" stroke="#ff5050" strokeWidth="1.4" strokeLinecap="round"/>
            </svg>
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: '#ff6b6b', lineHeight: 1.5 }}>
              {friendlyError(record.error ?? '')}
            </span>
          </div>
          <button
            disabled={retrying}
            onClick={async () => { setRetrying(true); await onRetry(record.id); setRetrying(false) }}
            style={{ alignSelf: 'flex-start', background: 'rgba(255,80,80,0.12)', border: '1px solid rgba(255,80,80,0.3)', borderRadius: '6px', padding: '5px 12px', cursor: retrying ? 'default' : 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '600', color: '#ff6b6b', opacity: retrying ? 0.5 : 1, display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            <svg width="11" height="11" viewBox="0 0 14 14" fill="none"><path d="M2 7a5 5 0 015-5 5 5 0 014.33 2.5M12 7a5 5 0 01-5 5 5 5 0 01-4.33-2.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/><path d="M11 2.5V5.5H8" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round"/></svg>
            {retrying ? 'Retrying…' : 'Retry Ingest'}
          </button>
        </div>
      )}

      <button onClick={() => onStartChat(record)} style={{ background: 'var(--accent)', border: 'none', color: '#070c16', padding: '11px 16px', borderRadius: '10px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'center', marginTop: 'auto' }}>
        <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
          <path d="M2 3A1.5 1.5 0 013.5 1.5h8A1.5 1.5 0 0113 3v6a1.5 1.5 0 01-1.5 1.5H8.5l-2 2v-2H3.5A1.5 1.5 0 012 9V3z" stroke="#070c16" strokeWidth="1.4"/>
        </svg>
        Start Conversation
      </button>
    </div>
  )
}

export default function RecordsPage() {
  const navigate = useNavigate()
  const { tokens } = useAuth()
  const { records, setRecords } = useOutletContext<AppContext>()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = records.find(r => r.id === selectedId) ?? null
  const [showAddRecord, setShowAddRecord] = useState(false)
  const [search, setSearch] = useState('')
  const [hasLoaded, setHasLoaded] = useState(false)
  const [error, setError] = useState('')
  const [refreshTick, setRefreshTick] = useState(0)

  const accessToken = tokens?.accessToken ?? ''

  const refreshRecords = useCallback(() => { setRefreshTick(t => t + 1) }, [])

  useEffect(() => {
    if (!accessToken) return
    let active = true
    listRecords(accessToken)
      .then(data => { if (active) { setRecords(data); setError(''); setHasLoaded(true) } })
      .catch(err => { if (active) { setError(err instanceof Error ? err.message : 'Failed to load records'); setHasLoaded(true) } })
    return () => { active = false }
  }, [accessToken, setRecords, refreshTick])

  const filtered = records.filter(r =>
    r.name.toLowerCase().includes(search.toLowerCase()) ||
    (r.url ?? '').toLowerCase().includes(search.toLowerCase())
  )

  async function handleUploadRecord(file: File) {
    if (!accessToken) throw new Error('Authentication token is missing')
    await uploadRecordFile(accessToken, file)
    refreshRecords()
  }

  async function handleRetryRecord(id: string) {
    if (!accessToken) return
    try {
      await retryRecordIngest(accessToken, id)
      refreshRecords()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry ingest')
    }
  }

  async function handleDeleteRecord(e: React.MouseEvent, id: string) {
    e.stopPropagation()
    if (!accessToken) return
    try {
      await deleteRecord(accessToken, id)
      if (selectedId === id) setSelectedId(null)
      refreshRecords()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete record')
    }
  }

  return (
    <div style={{ display: 'flex', flex: 1, overflow: 'hidden', height: '100%' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>

        <div style={{ padding: '24px 32px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
          <div>
            <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '22px', color: 'var(--text)', margin: 0, letterSpacing: '-0.02em' }}>Records</h1>
            <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', margin: '4px 0 0' }}>
              {records.length} records · {records.filter(r => r.status === 'indexed').length} indexed
            </p>
          </div>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
            <div style={{ position: 'relative' }}>
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', opacity: 0.4 }}>
                <circle cx="6" cy="6" r="4.5" stroke="var(--text)" strokeWidth="1.3"/>
                <path d="M9.5 9.5l2.5 2.5" stroke="var(--text)" strokeWidth="1.3" strokeLinecap="round"/>
              </svg>
              <input placeholder="Search records…" value={search} onChange={e => setSearch(e.target.value)} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '8px 12px 8px 34px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', outline: 'none', width: '200px' }} />
            </div>
            <button
              onClick={() => setShowAddRecord(true)}
              style={{ background: 'var(--accent)', border: 'none', color: '#070c16', padding: '9px 16px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '7px' }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v10M2 7h10" stroke="#070c16" strokeWidth="2" strokeLinecap="round"/></svg>
              Add Record
            </button>
            <div style={{ width: '1px', height: '20px', background: 'var(--border)' }} />
            <UserMenu />
          </div>
        </div>

        {error && (
          <div style={{ padding: '12px 32px 0', flexShrink: 0 }}>
            <ErrorBanner message={error} onDismiss={() => setError('')} />
          </div>
        )}

        {!hasLoaded && !error && (
          <div style={{ padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', flexShrink: 0 }}>
            Loading records…
          </div>
        )}

        <div style={{ display: 'flex', alignItems: 'center', padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', letterSpacing: '0.06em', flexShrink: 0 }}>
          <span style={{ flex: 1 }}>Record</span>
          <span style={{ width: '100px' }}>Status</span>
          <span style={{ width: '100px' }}>Details</span>
          <span style={{ width: '110px' }}>Added</span>
          <span style={{ width: '36px' }} />
        </div>

        <div style={{ overflowY: 'auto', flex: 1 }}>
          {filtered.map(record => (
            <div
              key={record.id}
              onClick={() => setSelectedId(selectedId === record.id ? null : record.id)}
              style={{ display: 'flex', alignItems: 'center', padding: '14px 32px', borderBottom: '1px solid var(--border)', cursor: 'pointer', transition: 'all 0.15s ease', gap: '8px', background: selectedId === record.id ? 'rgba(0,212,180,0.05)' : 'transparent', borderLeft: selectedId === record.id ? '2px solid var(--accent)' : '2px solid transparent' }}
              onMouseEnter={e => { if (selectedId !== record.id) (e.currentTarget as HTMLDivElement).style.background = 'rgba(255,255,255,0.03)' }}
              onMouseLeave={e => { if (selectedId !== record.id) (e.currentTarget as HTMLDivElement).style.background = 'transparent' }}
            >
              <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: '12px', minWidth: 0 }}>
                <RecordIcon record={record} />
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13.5px', color: 'var(--text)', fontWeight: '500', marginBottom: '3px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '380px' }}>{record.name}</div>
                  <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '380px' }}>{recordSubtext(record)}</div>
                </div>
              </div>
              <div style={{ width: '100px' }}><StatusBadge status={record.status} /></div>
              <div style={{ width: '100px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>{recordDetail(record)}</div>
              <div style={{ width: '110px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>{record.createdAt}</div>
              <button
                onClick={e => { void handleDeleteRecord(e, record.id) }}
                title="Delete record"
                style={{ width: '28px', height: '28px', background: 'none', border: '1px solid transparent', borderRadius: '6px', cursor: 'pointer', color: 'var(--text-dim)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, transition: 'all 0.15s' }}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = '#ff5050'; (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'transparent'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
              >
                <svg width="13" height="13" viewBox="0 0 14 14" fill="none">
                  <path d="M2 4h10M5 4V2.5h4V4M5.5 6.5v4M8.5 6.5v4M3 4l.7 7.5A1 1 0 004.7 12.5h4.6a1 1 0 001-.95L11 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </button>
            </div>
          ))}

          {filtered.length === 0 && hasLoaded && (
            <div style={{ textAlign: 'center', padding: '60px 32px', color: 'var(--text-dim)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
              {search ? 'No records match your search.' : 'No records yet. Upload a document to get started.'}
            </div>
          )}
        </div>
      </div>

      {selected && (
        <DetailPanel
          record={selected}
          onClose={() => setSelectedId(null)}
          onStartChat={r => navigate('/chat', { state: { source: r } })}
          onRetry={handleRetryRecord}
        />
      )}

      {showAddRecord && (
        <AddRecordModal
          onClose={() => setShowAddRecord(false)}
          onUploadFile={handleUploadRecord}
        />
      )}
    </div>
  )
}
