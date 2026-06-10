// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { cancelRecordIngest, createSource, deleteRecord, deleteSource, listRecords, listSources, retryRecordIngest, syncSource, updateGoogleSourceSelection, uploadRecordFile } from '@/lib/embedder/service'
import { imageIngestLabel, imageIngestStatusText, imageRecordSubtext } from '@/lib/embedder/image-ingest'
import type { AppContext, AppRecord, DriveSource, DriveSourceDraft } from '@/types'
import UserMenu from '@/components/UserMenu'
import AddRecordModal from '@/components/AddRecordModal'
import AddSourceModal from '@/components/AddSourceModal'
import EditSourceSelectionModal from '@/components/EditSourceSelectionModal'
import SourceProviderIcon from '@/components/SourceProviderIcon'
import { sourceProviderLabel } from '@/lib/embedder/source-provider'

const statusColors = {
  queued:     { bg: 'rgba(156,163,175,0.1)', color: '#9ca3af', dot: '#9ca3af' },
  indexed:    { bg: 'rgba(0,212,180,0.1)',  color: '#00d4b4', dot: '#00d4b4' },
  processing: { bg: 'rgba(255,180,0,0.1)',  color: '#ffb400', dot: '#ffb400' },
  failed:     { bg: 'rgba(255,80,80,0.1)',  color: '#ff5050', dot: '#ff5050' },
  cancelled:  { bg: 'rgba(156,163,175,0.1)', color: '#9ca3af', dot: '#9ca3af' },
}

const driveStatusColors = {
  active:       { bg: 'rgba(0,212,180,0.1)',   color: '#00d4b4', dot: '#00d4b4' },
  syncing:      { bg: 'rgba(255,180,0,0.1)',   color: '#ffb400', dot: '#ffb400' },
  error:        { bg: 'rgba(255,80,80,0.1)',   color: '#ff5050', dot: '#ff5050' },
  disconnected: { bg: 'rgba(156,163,175,0.1)', color: '#9ca3af', dot: '#9ca3af' },
}

const RECORD_POLL_INTERVAL_MS = 2500

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
    return imageRecordSubtext(record)
  }
  if (record.status === 'queued') return 'waiting for indexing…'
  if (record.status === 'processing') return 'indexing…'
  if (record.chunks != null) {
    return record.pages != null ? `${record.chunks} chunks · ${record.pages} pages` : `${record.chunks} chunks`
  }
  return record.pages != null ? `${record.pages} pages` : 'pending'
}

function recordDetail(record: AppRecord): string {
  if (record.format === 'link') {
    try { return new URL(record.url ?? '').hostname } catch { return '—' }
  }
  return record.size ?? '—'
}

function formatInlineError(err: string): string {
	const normalized = err.replace(/\s+/g, ' ').trim()
	return `"${normalized}"`
}

const integrationActionButtonBase: React.CSSProperties = {
  borderRadius: '8px',
  padding: '7px 10px',
  fontFamily: 'JetBrains Mono, monospace',
  fontSize: '10px',
  lineHeight: 1.2,
  cursor: 'pointer',
  border: '1px solid var(--border)',
  background: 'rgba(255,255,255,0.04)',
  color: 'var(--text)',
  minWidth: '78px',
}

const integrationDeleteButtonStyle: React.CSSProperties = {
	...integrationActionButtonBase,
	background: 'rgba(255,80,80,0.08)',
	border: '1px solid rgba(255,80,80,0.25)',
	color: '#ff7b7b',
	minWidth: '70px',
}

function DetailPanel({ record, onClose, onStartChat }: { record: AppRecord; onClose: () => void; onStartChat: (r: AppRecord) => void }) {
  const meta: { label: string; value: string }[] = []
  if (record.format !== 'link' && record.size) meta.push({ label: 'SIZE', value: record.size })
  meta.push({ label: 'ADDED', value: record.createdAt })
  if (record.format !== 'link' && record.format !== 'image' && record.pages != null)
    meta.push({ label: 'PAGES', value: String(record.pages) })
  if (record.format === 'image') meta.push({ label: 'MODE', value: imageIngestLabel(record) })
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
          <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>
            {record.format === 'image' ? imageIngestStatusText(record) : 'Fully indexed · available for retrieval'}
          </span>
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

export default function HomePage() {
  const navigate = useNavigate()
  const { tokens } = useAuth()
  const { records, setRecords, driveSources, setDriveSources, activeDomain } = useOutletContext<AppContext>()
  const [selected, setSelected] = useState<AppRecord | null>(null)
  const [showAddRecord, setShowAddRecord] = useState(false)
  const [showAddSource, setShowAddSource] = useState(false)
  const [search, setSearch] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [syncingSourceIDs, setSyncingSourceIDs] = useState<string[]>([])
  const [sourceSyncNotice, setSourceSyncNotice] = useState<Record<string, { kind: 'info' | 'error'; text: string }>>({})
  const [expandedSourceErrors, setExpandedSourceErrors] = useState<string[]>([])
  const [editingSource, setEditingSource] = useState<DriveSource | null>(null)

  const accessToken = tokens?.accessToken ?? ''
  const domainID = activeDomain?.id ?? ''
  const cachedGoogleSource = driveSources.find(source => source.sourceType === 'google_drive' && !!source.accessToken)

  const refreshData = useCallback(async (options?: { silent?: boolean }) => {
    if (!accessToken) {
      setRecords([])
      setDriveSources([])
      return
    }

    if (!options?.silent) setIsLoading(true)
    try {
      const [nextSources, nextRecords] = await Promise.all([
        listSources(accessToken, domainID),
        listRecords(accessToken, domainID),
      ])
      setDriveSources(nextSources)
      setRecords(nextRecords)
      setLoadError('')
    } catch (err) {
      console.error('failed loading records/sources', err)
      setLoadError(err instanceof Error ? err.message : 'Failed to load records and sources')
    } finally {
      if (!options?.silent) setIsLoading(false)
    }
  }, [accessToken, domainID, setDriveSources, setRecords])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void refreshData()
  }, [refreshData])

  useEffect(() => {
    if (!accessToken || !records.some(record => record.status === 'queued' || record.status === 'processing')) return
    const timer = window.setInterval(() => { void refreshData({ silent: true }) }, RECORD_POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [accessToken, records, refreshData])

  useEffect(() => {
    if (selected && !records.some(record => record.id === selected.id)) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSelected(null)
    }
  }, [records, selected])

  const filtered = records.filter(s =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    (s.url ?? '').toLowerCase().includes(search.toLowerCase())
  )

  async function handleAddSource(source: DriveSourceDraft) {
    if (!accessToken) {
      throw new Error('Authentication token is missing')
    }

    const created = await createSource(accessToken, domainID, source)
    try {
      await syncSource(accessToken, domainID, created.id)
    } catch (err) {
      console.error('source sync failed', err)
      setLoadError(
        err instanceof Error
          ? `Source created, but sync failed: ${err.message}`
          : 'Source created, but sync failed',
      )
    }

    await refreshData()
  }

  async function handleUploadRecord(file: File) {
    if (!accessToken) {
      throw new Error('Authentication token is missing')
    }

    await uploadRecordFile(accessToken, domainID, file)
    await refreshData()
  }

  async function handleDeleteRecord(e: React.MouseEvent, id: string) {
    e.stopPropagation()
    if (!accessToken) return
    await deleteRecord(accessToken, domainID, id)
    if (selected?.id === id) setSelected(null)
    await refreshData()
  }

  async function handleRetryRecordIngest(record: AppRecord) {
    if (!accessToken) return
    try {
      await retryRecordIngest(accessToken, domainID, record.id)
      setRecords(prev => prev.map(item => (
        item.id === record.id ? { ...item, status: 'queued', error: undefined } : item
      )))
      setLoadError('')
      await refreshData()
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : 'Failed to retry ingest')
    }
  }

  async function handleCancelRecordIngest(record: AppRecord) {
    if (!accessToken) return
    try {
      await cancelRecordIngest(accessToken, domainID, record.id)
      setRecords(prev => prev.map(item => (
        item.id === record.id ? { ...item, status: 'cancelled', error: undefined } : item
      )))
      setLoadError('')
      await refreshData()
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : 'Failed to cancel ingest')
    }
  }

  async function handleRetrySourceSync(sourceID: string) {
    if (!accessToken) return
    setSyncingSourceIDs(prev => (prev.includes(sourceID) ? prev : [...prev, sourceID]))
    setSourceSyncNotice(prev => ({
      ...prev,
      [sourceID]: { kind: 'info', text: 'Sync in progress...' },
    }))
    try {
      const res = await syncSource(accessToken, domainID, sourceID)
      setLoadError('')
      setSourceSyncNotice(prev => ({
        ...prev,
        [sourceID]: {
          kind: 'info',
          text: `Sync complete: supported files discovered ${res.discovered}, queued ${res.queued}, updated ${res.updated}, unchanged ${res.unchanged}.`,
        },
      }))
      await refreshData()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to sync source'
      setSourceSyncNotice(prev => ({
        ...prev,
        [sourceID]: { kind: 'error', text: message },
      }))
    } finally {
      setSyncingSourceIDs(prev => prev.filter(id => id !== sourceID))
    }
  }

  async function handleDeleteSource(sourceID: string) {
    if (!accessToken) return
    if (!window.confirm('Delete this source?')) return
    try {
      await deleteSource(accessToken, domainID, sourceID)
      setLoadError('')
      await refreshData()
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : 'Failed to delete source')
    }
  }

  async function handleSaveSourceSelection(source: DriveSource, selectedFileIDs: string[], selectedFolderIDs: string[]) {
    if (!accessToken) return
    try {
      await updateGoogleSourceSelection(accessToken, domainID, source.id, selectedFileIDs, selectedFolderIDs)
      await handleRetrySourceSync(source.id)
      setLoadError('')
      await refreshData()
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : 'Failed to update source selection')
      throw err
    }
  }

  async function handleCopyError(error: string) {
    try {
      await navigator.clipboard.writeText(error)
    } catch {
      setLoadError('Failed to copy sync error to clipboard')
    }
  }

  function handleBrowseSourceRecords(sourceID: string, sourceName: string) {
    navigate('/records', { state: { sourceID, sourceName } })
  }

  return (
    <div style={{ display: 'flex', flex: 1, overflow: 'hidden', height: '100%' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>

        {/* Page header */}
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
            <button
              onClick={() => setShowAddSource(true)}
              style={{ background: 'var(--accent)', border: 'none', color: '#070c16', padding: '9px 16px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '7px' }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v10M2 7h10" stroke="#070c16" strokeWidth="2" strokeLinecap="round"/></svg>
              Add Source
            </button>
            <div style={{ width: '1px', height: '20px', background: 'var(--border)' }} />
            <UserMenu />
          </div>
        </div>

        {(isLoading || loadError) && (
          <div style={{ padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: loadError ? '#ff6b6b' : 'var(--text-dim)' }}>
            {loadError || 'Loading records and sources...'}
          </div>
        )}

        {/* Column headers */}
        <div style={{ display: 'flex', alignItems: 'center', padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', letterSpacing: '0.06em', flexShrink: 0 }}>
          <span style={{ flex: 1 }}>Record</span>
          <span style={{ width: '100px' }}>Status</span>
          <span style={{ width: '100px' }}>Details</span>
          <span style={{ width: '110px' }}>Added</span>
          <span style={{ width: '220px', textAlign: 'right' }}>Actions</span>
        </div>

        {/* List */}
        <div style={{ overflowY: 'auto', flex: 1 }}>
          {filtered.map(record => (
            <div
              key={record.id}
              onClick={() => setSelected(selected?.id === record.id ? null : record)}
              style={{ display: 'flex', flexDirection: 'column', padding: '14px 32px', borderBottom: '1px solid var(--border)', cursor: 'pointer', transition: 'all 0.15s ease', gap: '8px', background: selected?.id === record.id ? 'rgba(0,212,180,0.05)' : 'transparent', borderLeft: selected?.id === record.id ? '2px solid var(--accent)' : '2px solid transparent' }}
              onMouseEnter={e => { if (selected?.id !== record.id) (e.currentTarget as HTMLDivElement).style.background = 'rgba(255,255,255,0.03)' }}
              onMouseLeave={e => { if (selected?.id !== record.id) (e.currentTarget as HTMLDivElement).style.background = 'transparent' }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: '12px', minWidth: 0 }}>
                  <RecordIcon record={record} />
                  <div style={{ minWidth: 0 }}>
                    <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13.5px', color: 'var(--text)', fontWeight: '500', marginBottom: '3px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{record.name}</div>
                    <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{recordSubtext(record)}</div>
                  </div>
                </div>
                <div style={{ width: '100px' }}><StatusBadge status={record.status} /></div>
                <div style={{ width: '100px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>{recordDetail(record)}</div>
                <div style={{ width: '110px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>{record.createdAt}</div>
                <div style={{ width: '220px', display: 'flex', justifyContent: 'flex-end', gap: '6px' }} onClick={e => e.stopPropagation()}>
                  {record.status === 'failed' && (
                    <button
                      onClick={() => { void handleRetryRecordIngest(record) }}
                      style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '6px', padding: '6px 8px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                    >
                      Retry ingest
                    </button>
                  )}
                  {record.status === 'cancelled' && (
                    <button
                      onClick={() => { void handleRetryRecordIngest(record) }}
                      style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '6px', padding: '6px 8px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                    >
                      Retry ingest
                    </button>
                  )}
                  {record.status === 'processing' && (
                    <button
                      onClick={() => { void handleCancelRecordIngest(record) }}
                      style={{ background: 'rgba(255,180,0,0.08)', border: '1px solid rgba(255,180,0,0.25)', borderRadius: '6px', padding: '6px 8px', color: '#ffb400', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                    >
                      Cancel ingest
                    </button>
                  )}
                  {record.error && (
                    <button
                      onClick={() => { void handleCopyError(record.error ?? '') }}
                      style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '6px', padding: '6px 8px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                    >
                      Copy error
                    </button>
                  )}
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
              </div>
              {record.error && (
                <div style={{ paddingLeft: '48px' }}>
                  <div
                    title={record.error}
                    style={{
                      display: 'inline-block',
                      minWidth: '260px',
                      maxWidth: 'min(520px, 100%)',
                      fontFamily: 'JetBrains Mono, monospace',
                      fontSize: '10px',
                      color: '#ff8080',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      background: 'rgba(255,80,80,0.08)',
                      border: '1px solid rgba(255,80,80,0.22)',
                      borderRadius: '8px',
                      padding: '6px 10px',
                    }}
                  >
                    {formatInlineError(record.error)}
                  </div>
                </div>
              )}
            </div>
          ))}

          {/* Drive integrations */}
          {driveSources.length > 0 && (
            <>
              <div style={{ padding: '10px 32px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', letterSpacing: '0.06em' }}>INTEGRATIONS</span>
                <div style={{ flex: 1, height: '1px', background: 'var(--border)' }} />
              </div>
              {driveSources.map(ds => {
                const c = driveStatusColors[ds.status] ?? driveStatusColors.active
                const sourceKind = sourceProviderLabel(ds.sourceType)
                const canEditSelection = ds.sourceType === 'google_drive'
                const isSyncing = ds.status === 'syncing' || syncingSourceIDs.includes(ds.id)
                const syncLabel = ds.status === 'error' || ds.status === 'disconnected'
                  ? 'Retry sync'
                  : isSyncing
                    ? 'Syncing...'
                    : 'Sync now'
                const syncNotice = sourceSyncNotice[ds.id]
                const sourceErrorText = syncNotice?.kind === 'error'
                  ? syncNotice.text
                  : (isSyncing ? '' : (ds.lastSyncError ?? ''))
                const sourceInfoText = syncNotice?.kind === 'info' ? syncNotice.text : ''
                const isErrorExpanded = expandedSourceErrors.includes(ds.id)
                return (
                  <div
                    key={ds.id}
                    onClick={() => handleBrowseSourceRecords(ds.id, ds.name)}
                    style={{ display: 'flex', flexDirection: 'column', padding: '14px 32px', borderBottom: '1px solid var(--border)', gap: '8px', borderLeft: '2px solid transparent', cursor: 'pointer' }}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', width: '100%' }}>
                      <div style={{ flex: 1, display: 'flex', alignItems: 'flex-start', gap: '12px', minWidth: 0 }}>
                        <SourceProviderIcon sourceType={ds.sourceType} framed />
                        <div style={{ minWidth: 0, width: '100%' }}>
                          <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13.5px', color: 'var(--text)', fontWeight: '500', marginBottom: '3px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '380px' }}>{ds.name}</div>
                          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
                            {sourceKind}{ds.syncEnabled ? ` · syncs every ${ds.autoSyncInterval}m` : ' · manual sync'}{ds.lastSyncAt ? ` · last sync ${ds.lastSyncAt}` : ''}
                          </div>
                        </div>
                      </div>
                      <div style={{ width: '100px' }}>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', background: c.bg, color: c.color, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '3px 8px', borderRadius: '20px', fontWeight: '500' }}>
                          <span style={{ width: '5px', height: '5px', borderRadius: '50%', background: c.dot }} />
                          {ds.status}
                        </span>
                      </div>
                      <div style={{ width: '360px', display: 'flex', gap: '6px', justifyContent: 'flex-end', flexWrap: 'wrap' }} onClick={e => e.stopPropagation()}>
                        {canEditSelection && (
                          <button
                            onClick={() => setEditingSource(ds)}
                            style={integrationActionButtonBase}
                          >
                            Edit
                          </button>
                        )}
                        <button
                          onClick={() => { void handleRetrySourceSync(ds.id) }}
                          disabled={isSyncing}
                          style={{
                            ...integrationActionButtonBase,
                            opacity: isSyncing ? 0.6 : 1,
                            cursor: isSyncing ? 'default' : 'pointer',
                          }}
                        >
                          {syncLabel}
                        </button>
                        <button
                          onClick={() => { void handleDeleteSource(ds.id) }}
                          style={integrationDeleteButtonStyle}
                        >
                          Delete
                        </button>
                      </div>
                      <div style={{ width: '110px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>{ds.createdAt}</div>
                    </div>
                    {sourceErrorText && (
                      <div style={{ width: '100%' }} onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                          <button
                            onClick={() => setExpandedSourceErrors(prev => prev.includes(ds.id) ? prev.filter(id => id !== ds.id) : [...prev, ds.id])}
                            style={{
                              width: 'auto',
                              textAlign: 'left',
                              background: 'rgba(255,80,80,0.08)',
                              border: '1px solid rgba(255,80,80,0.22)',
                              borderRadius: '6px',
                              padding: '6px 8px',
                              boxSizing: 'border-box',
                              color: '#ff8080',
                              cursor: 'pointer',
                              fontFamily: 'JetBrains Mono, monospace',
                              fontSize: '10px',
                              display: 'inline-flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              gap: '8px',
                            }}
                          >
                            <span>{isErrorExpanded ? 'Hide error details' : 'Show error details'}</span>
                            <span style={{ opacity: 0.8 }}>{isErrorExpanded ? '▲' : '▼'}</span>
                          </button>
                          <button
                            title="Copy error"
                            onClick={() => { void handleCopyError(sourceErrorText) }}
                            style={{
                              width: '28px',
                              height: '28px',
                              borderRadius: '6px',
                              border: '1px solid rgba(255,80,80,0.22)',
                              background: 'rgba(255,80,80,0.08)',
                              color: '#ff8080',
                              cursor: 'pointer',
                              display: 'inline-flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              padding: 0,
                            }}
                          >
                            <svg width="13" height="13" viewBox="0 0 14 14" fill="none">
                              <rect x="5" y="2" width="7" height="9" rx="1.4" stroke="currentColor" strokeWidth="1.2" />
                              <path d="M3 9.5H2.8A1.3 1.3 0 0 1 1.5 8.2V3.8A1.3 1.3 0 0 1 2.8 2.5H7" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                            </svg>
                          </button>
                        </div>
                        {isErrorExpanded && (
                          <div
                            title={sourceErrorText}
                            style={{
                              marginTop: '6px',
                              fontFamily: 'JetBrains Mono, monospace',
                              fontSize: '10px',
                              color: '#ff8080',
                              whiteSpace: 'pre-wrap',
                              overflowWrap: 'anywhere',
                              lineHeight: 1.45,
                              width: '100%',
                              background: 'rgba(255,80,80,0.08)',
                              border: '1px solid rgba(255,80,80,0.22)',
                              borderRadius: '6px',
                              padding: '6px 8px',
                              boxSizing: 'border-box',
                            }}
                          >
                            {formatInlineError(sourceErrorText)}
                          </div>
                        )}
                      </div>
                    )}
                    {sourceInfoText && (
                      <div
                        title={sourceInfoText}
                        style={{
                          fontFamily: 'JetBrains Mono, monospace',
                          fontSize: '10px',
                          color: 'var(--accent)',
                          whiteSpace: 'pre-wrap',
                          overflowWrap: 'anywhere',
                          lineHeight: 1.45,
                          width: '100%',
                          background: 'rgba(0,212,180,0.08)',
                          border: '1px solid rgba(0,212,180,0.22)',
                          borderRadius: '6px',
                          padding: '6px 8px',
                          boxSizing: 'border-box',
                        }}
                      >
                        {sourceInfoText}
                      </div>
                    )}
                  </div>
                )
              })}
            </>
          )}

        </div>
      </div>

      {selected && (
        <DetailPanel
          record={selected}
          onClose={() => setSelected(null)}
          onStartChat={r => navigate('/chat', { state: { source: r } })}
        />
      )}

      {showAddRecord && (
        <AddRecordModal
          onClose={() => setShowAddRecord(false)}
          onUploadFile={handleUploadRecord}
        />
      )}

      {showAddSource && (
        <AddSourceModal
          authToken={accessToken}
          initialGoogleAccessToken={cachedGoogleSource?.accessToken}
          initialGoogleRefreshToken={cachedGoogleSource?.refreshToken}
          onClose={() => setShowAddSource(false)}
          onAdd={handleAddSource}
        />
      )}

      {editingSource && (
        <EditSourceSelectionModal
          authToken={accessToken}
          source={editingSource}
          onClose={() => setEditingSource(null)}
          onSave={async (selectedFileIDs, selectedFolderIDs) => {
            await handleSaveSourceSelection(editingSource, selectedFileIDs, selectedFolderIDs)
            setEditingSource(null)
          }}
        />
      )}
    </div>
  )
}
