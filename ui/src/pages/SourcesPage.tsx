import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { createSource, deleteSource, listSources, syncSource, updateGoogleSourceSelection } from '@/lib/embedder/service'
import type { AppContext, DriveSource, DriveSourceDraft } from '@/types'
import UserMenu from '@/components/UserMenu'
import AddSourceModal from '@/components/AddSourceModal'
import EditSourceSelectionModal from '@/components/EditSourceSelectionModal'

const driveStatusColors = {
  active:       { bg: 'rgba(0,212,180,0.1)',   color: '#00d4b4', dot: '#00d4b4' },
  syncing:      { bg: 'rgba(255,180,0,0.1)',   color: '#ffb400', dot: '#ffb400' },
  error:        { bg: 'rgba(255,80,80,0.1)',   color: '#ff5050', dot: '#ff5050' },
  disconnected: { bg: 'rgba(156,163,175,0.1)', color: '#9ca3af', dot: '#9ca3af' },
}

function DriveIcon() {
  return (
    <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: 'rgba(66,133,244,0.12)', border: '1px solid rgba(66,133,244,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
        <path d="M8.5 3H15.5L22 13H16L13 8.5L10 13H2L8.5 3Z" fill="#4285f4" opacity="0.85"/>
        <path d="M2 13L5.5 19H18.5L22 13H16L13 18H11L8 13H2Z" fill="#34a853" opacity="0.85"/>
        <path d="M10 13L13 8.5L16 13H10Z" fill="#fbbc04"/>
      </svg>
    </div>
  )
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

function SourceRow(
  {
    source,
    syncing,
    onEdit,
    onSync,
    onOpen,
    onDelete,
  }: {
    source: DriveSource
    syncing: boolean
    onEdit: () => void
    onSync: () => void
    onOpen: () => void
    onDelete: () => void
  },
) {
  const c = driveStatusColors[source.status] ?? driveStatusColors.active
  const sourceKindLabel = source.sourceType === 'google_drive' ? 'Google Drive' : 'Rclone'
  const sourceLocation = source.sourceType === 'google_drive'
    ? (source.selectedFileIDs.length > 0
      ? `${source.selectedFileIDs.length} selected files`
      : source.selectedFolderIDs.length > 0
        ? `${source.selectedFolderIDs.length} selected folders`
        : 'full scope')
    : source.rcloneRootPath || source.rcloneScopePaths?.join(', ') || '—'
  return (
    <div style={{ display: 'flex', alignItems: 'center', padding: '14px 32px', borderBottom: '1px solid var(--border)', gap: '8px', borderLeft: '2px solid transparent' }}>
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: '12px', minWidth: 0 }}>
        <DriveIcon />
        <div style={{ minWidth: 0 }}>
          <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13.5px', color: 'var(--text)', fontWeight: '500', marginBottom: '3px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '340px' }}>{source.name}</div>
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
            {sourceKindLabel} · scope: {sourceLocation}{source.syncEnabled ? ` · syncs every ${source.autoSyncInterval}m` : ' · manual sync'}
          </div>
          {source.lastSyncError && (
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff6b6b', marginTop: '4px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '340px' }}>
              {source.lastSyncError}
            </div>
          )}
        </div>
      </div>

      <div style={{ width: '110px' }}>
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', background: c.bg, color: c.color, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '3px 8px', borderRadius: '20px', fontWeight: '500' }}>
          <span style={{ width: '5px', height: '5px', borderRadius: '50%', background: c.dot, ...(source.status === 'syncing' ? { animation: 'pulse 1.5s ease-in-out infinite' } : {}) }} />
          {source.status}
        </span>
      </div>

      <div style={{ width: '110px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>{source.createdAt}</div>

      <button
        onClick={onOpen}
        title="Open in prompt"
        style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: 'none', border: '1px solid var(--border)', borderRadius: '6px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', transition: 'all 0.15s', flexShrink: 0 }}
        onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--accent)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)' }}
        onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
      >
        Open
      </button>

      {source.sourceType === 'google_drive' && (
        <button
          onClick={onEdit}
          title="Edit source selection"
          style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: 'none', border: '1px solid var(--border)', borderRadius: '6px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', transition: 'all 0.15s', flexShrink: 0 }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--accent)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
        >
          Edit
        </button>
      )}

      <button
        onClick={onSync}
        disabled={syncing}
        title="Sync now"
        style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: 'none', border: '1px solid var(--border)', borderRadius: '6px', cursor: syncing ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', opacity: syncing ? 0.5 : 1, transition: 'all 0.15s', flexShrink: 0 }}
        onMouseEnter={e => { if (!syncing) { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--accent)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--accent)' } }}
        onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
      >
        <svg width="11" height="11" viewBox="0 0 14 14" fill="none" style={{ ...(syncing ? { animation: 'spin 1s linear infinite' } : {}) }}>
          <path d="M12 7A5 5 0 112 7" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
          <path d="M12 3v4h-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
        {syncing ? 'Syncing…' : 'Sync now'}
      </button>

      <button
        onClick={onDelete}
        title="Delete source"
        style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: 'none', border: '1px solid rgba(255,80,80,0.4)', borderRadius: '6px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff8a8a', transition: 'all 0.15s', flexShrink: 0 }}
        onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = '#ff5050'; (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' }}
        onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'rgba(255,80,80,0.4)'; (e.currentTarget as HTMLButtonElement).style.color = '#ff8a8a' }}
      >
        Delete
      </button>
    </div>
  )
}

export default function SourcesPage() {
  const navigate = useNavigate()
  const { tokens } = useAuth()
  const { driveSources, setDriveSources } = useOutletContext<AppContext>()
  const [showAddSource, setShowAddSource] = useState(false)
  const [editingSource, setEditingSource] = useState<DriveSource | null>(null)
  const [hasLoaded, setHasLoaded] = useState(false)
  const [error, setError] = useState('')
  const [syncing, setSyncing] = useState<Record<string, boolean>>({})

  const accessToken = tokens?.accessToken ?? ''

  const [refreshTick, setRefreshTick] = useState(0)
  const refreshSources = useCallback(() => { setRefreshTick(t => t + 1) }, [])
  const cachedGoogleSource = driveSources.find(source => source.sourceType === 'google_drive' && !!source.accessToken)

  useEffect(() => {
    if (!accessToken) return
    let active = true
    listSources(accessToken)
      .then(data => { if (active) { setDriveSources(data); setError(''); setHasLoaded(true) } })
      .catch(err => { if (active) { setError(err instanceof Error ? err.message : 'Failed to load sources'); setHasLoaded(true) } })
    return () => { active = false }
  }, [accessToken, setDriveSources, refreshTick])

  async function handleAddSource(source: DriveSourceDraft) {
    if (!accessToken) throw new Error('Authentication token is missing')
    const created = await createSource(accessToken, source)
    refreshSources()
    setSyncing(prev => ({ ...prev, [created.id]: true }))
    void syncSource(accessToken, created.id)
      .then(() => {
        refreshSources()
      })
      .catch((err) => {
        setError(`Source created, but initial sync failed: ${err instanceof Error ? err.message : 'unknown error'}`)
      })
      .finally(() => {
        setSyncing(prev => ({ ...prev, [created.id]: false }))
      })
  }

  async function handleSync(id: string) {
    if (!accessToken || syncing[id]) return
    setSyncing(prev => ({ ...prev, [id]: true }))
    try {
      await syncSource(accessToken, id)
      refreshSources()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed')
    } finally {
      setSyncing(prev => ({ ...prev, [id]: false }))
    }
  }

  async function handleDelete(id: string) {
    if (!accessToken) return
    if (!window.confirm('Delete this source?')) return
    try {
      await deleteSource(accessToken, id)
      refreshSources()
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  async function handleSaveSourceSelection(source: DriveSource, selectedFileIDs: string[]) {
    if (!accessToken) return
    try {
      await updateGoogleSourceSelection(accessToken, source.id, selectedFileIDs, source.selectedFolderIDs ?? [])
      await handleSync(source.id)
      refreshSources()
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update source selection')
      throw err
    }
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>

      <div style={{ padding: '24px 32px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '22px', color: 'var(--text)', margin: 0, letterSpacing: '-0.02em' }}>Sources</h1>
          <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', margin: '4px 0 0' }}>
            {driveSources.length} {driveSources.length === 1 ? 'integration' : 'integrations'}
          </p>
        </div>
        <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
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

      {error && (
        <div style={{ padding: '12px 32px 0', flexShrink: 0 }}>
          <ErrorBanner message={error} onDismiss={() => setError('')} />
        </div>
      )}

      {!hasLoaded && !error && (
        <div style={{ padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', flexShrink: 0 }}>
          Loading sources…
        </div>
      )}

        <div style={{ display: 'flex', alignItems: 'center', padding: '10px 32px', borderBottom: '1px solid var(--border)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', letterSpacing: '0.06em', flexShrink: 0 }}>
          <span style={{ flex: 1 }}>Source</span>
          <span style={{ width: '110px' }}>Status</span>
          <span style={{ width: '110px' }}>Added</span>
          <span style={{ width: '70px' }} />
          <span style={{ width: '70px' }} />
          <span style={{ width: '90px' }} />
        </div>

      <div style={{ overflowY: 'auto', flex: 1 }}>
        {driveSources.map(source => (
          <SourceRow
            key={source.id}
            source={source}
            syncing={!!syncing[source.id]}
            onEdit={() => setEditingSource(source)}
            onOpen={() => navigate('/chat', { state: { sourceID: source.id, sourceName: source.name } })}
            onSync={() => { void handleSync(source.id) }}
            onDelete={() => { void handleDelete(source.id) }}
          />
        ))}

        {driveSources.length === 0 && hasLoaded && (
          <div style={{ textAlign: 'center', padding: '60px 32px', color: 'var(--text-dim)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px' }}>
            No sources yet. Connect a source to start syncing documents.
          </div>
        )}
      </div>

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
          onSave={async selectedFileIDs => {
            await handleSaveSourceSelection(editingSource, selectedFileIDs)
            setEditingSource(null)
          }}
        />
      )}
    </div>
  )
}
