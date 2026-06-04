// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { useOutletContext, useLocation } from 'react-router-dom'
import type { AppContext, AppRecord, ChatMessage, Conversation, MsgSource } from '@/types'
import UserMenu from '@/components/UserMenu'
import { useAuth } from '@/hooks/useAuth'
import { listOllamaModels, streamChat } from '@/lib/api'
import type { Citation } from '@/lib/api'
import { loadModelConfig, toBackendModelConfig } from '@/lib/modelConfig'
import type { ModelConfig } from '@/lib/modelConfig'
import { deleteConversation, getConversation, listRecordsBySource, syncSource } from '@/lib/embedder/service'

interface ChatRouteState {
  source?: AppRecord
  sourceID?: string
  sourceName?: string
}


function formatInlineError(err: string): string {
  const normalized = err.replace(/\s+/g, ' ').trim()
  return `"${normalized}"`
}



function citationToSource(c: Citation): MsgSource {
  return {
    id: `${c.record_id}-${c.chunk_index}`,
    recordID: c.record_id,
    doc: c.record_name,
    page: c.chunk_index + 1,
    excerpt: c.excerpt,
    url: c.external_url,
  }
}

function citationCounts(citations: MsgSource[]) {
  const records = new Set(citations.map(citation => citation.recordID || citation.doc))
  return { records: records.size, chunks: citations.length }
}

type ModelStatus = 'checking' | 'connected' | 'configured' | 'model-unavailable' | 'provider-unavailable' | 'server-default'

function initialModelStatus(config: ModelConfig): ModelStatus {
  if (!toBackendModelConfig(config)) return 'server-default'
  return config.provider === 'local' ? 'checking' : 'configured'
}

function modelStatusDetails(config: ModelConfig, status: ModelStatus) {
  const selectedProvider = config.provider === 'local' ? 'Ollama' : config.provider === 'openai' ? 'OpenAI' : 'Anthropic'
  const provider = status === 'server-default' ? 'Server default' : selectedProvider
  const model = status === 'server-default' ? '' : config.model
  const labels: Record<ModelStatus, string> = {
    checking: 'Checking',
    connected: 'Connected',
    configured: 'Configured',
    'model-unavailable': 'Model unavailable',
    'provider-unavailable': 'Provider unavailable',
    'server-default': 'In use',
  }
  const colors: Record<ModelStatus, string> = {
    checking: 'var(--text-dim)',
    connected: 'var(--accent)',
    configured: 'var(--accent)',
    'model-unavailable': '#ffb400',
    'provider-unavailable': '#ff5050',
    'server-default': '#ffb400',
  }
  return { provider, model, label: labels[status], color: colors[status] }
}

function renderMarkdown(text: string) {
  return text.split('\n').map((line, i) => {
    if (line.startsWith('- ')) {
      const content = line.slice(2).replace(/\*\*([^*]+)\*\*/g, (_, t) => `<strong style="color:var(--text);font-weight:600">${t}</strong>`)
      return <div key={i} style={{ display: 'flex', gap: '8px', marginBottom: '4px' }}><span style={{ color: 'var(--accent)', marginTop: '2px', flexShrink: 0 }}>·</span><span dangerouslySetInnerHTML={{ __html: content }} /></div>
    }
    if (line === '') return <div key={i} style={{ height: '8px' }} />
    const content = line.replace(/\*\*([^*]+)\*\*/g, (_, t) => `<strong style="color:var(--text);font-weight:600">${t}</strong>`)
    return <div key={i} dangerouslySetInnerHTML={{ __html: content }} />
  })
}

function MessageBubble({ msg, onShowSources }: { msg: ChatMessage; onShowSources?: (sources: MsgSource[]) => void }) {
  if (msg.role === 'user') {
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '20px' }}>
        <div style={{ background: 'rgba(255,255,255,0.06)', border: '1px solid var(--border)', borderRadius: '14px 14px 4px 14px', padding: '12px 16px', maxWidth: '520px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text)', lineHeight: 1.6 }}>{msg.content}</div>
      </div>
    )
  }
  const counts = citationCounts(msg.sources ?? [])
  return (
    <div style={{ display: 'flex', gap: '12px', marginBottom: '24px', alignItems: 'flex-start' }}>
      <div style={{ width: '28px', height: '28px', borderRadius: '8px', background: 'rgba(0,212,180,0.15)', border: '1px solid rgba(0,212,180,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, marginTop: '2px' }}>
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 3h3v3H3zM8 3h3v3H8zM3 8h3v3H3z" fill="var(--accent)" opacity="0.8"/><path d="M8 8h3v3H8z" fill="var(--accent)" opacity="0.25"/></svg>
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', lineHeight: 1.7 }}>{renderMarkdown(msg.content)}</div>
        {msg.sources && msg.sources.length > 0 && (
          <div style={{ marginTop: '10px' }}>
            <button
              onClick={() => onShowSources?.(msg.sources!)}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', background: 'rgba(0,212,180,0.07)', border: '1px solid rgba(0,212,180,0.2)', borderRadius: '6px', padding: '4px 10px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--accent)', transition: 'all 0.15s' }}
              onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(0,212,180,0.13)' }}
              onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(0,212,180,0.07)' }}
            >
              <svg width="11" height="11" viewBox="0 0 14 14" fill="none"><rect x="1" y="1" width="9" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.3"/><path d="M4 5h5M4 8h3" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round"/></svg>
              Records · {counts.records} · Chunks · {counts.chunks}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function TypingIndicator() {
  return (
    <div style={{ display: 'flex', gap: '12px', marginBottom: '24px', alignItems: 'flex-start' }}>
      <div style={{ width: '28px', height: '28px', borderRadius: '8px', background: 'rgba(0,212,180,0.15)', border: '1px solid rgba(0,212,180,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 3h3v3H3zM8 3h3v3H8zM3 8h3v3H8z" fill="var(--accent)" opacity="0.8"/><path d="M8 8h3v3H8z" fill="var(--accent)" opacity="0.25"/></svg>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: '5px', paddingTop: '6px' }}>
        {[0, 1, 2].map(i => <div key={i} style={{ width: '6px', height: '6px', borderRadius: '50%', background: 'var(--text-dim)', animation: `bounce 1.2s ease-in-out ${i * 0.2}s infinite` }} />)}
      </div>
    </div>
  )
}

function ConversationList({
  conversations,
  activeId,
  onSelect,
  onNew,
  onDelete,
}: {
  conversations: Conversation[]
  activeId: string | null
  onSelect: (id: string) => void
  onNew: () => void
  onDelete: (id: string) => void
}) {
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const filteredConversations = searchQuery.trim()
    ? conversations.filter(c => (c.title || 'Untitled').toLowerCase().includes(searchQuery.toLowerCase()))
    : conversations

  return (
    <div style={{ width: '220px', minWidth: '220px', height: '100%', borderRight: '1px solid var(--border)', display: 'flex', flexDirection: 'column', background: 'var(--sidebar-bg)' }}>
      <div style={{ padding: '16px 12px 8px', flexShrink: 0, display: 'flex', flexDirection: 'column', gap: '8px' }}>
        <button
          onClick={onNew}
          style={{ width: '100%', display: 'flex', alignItems: 'center', gap: '8px', padding: '9px 12px', background: 'rgba(0,212,180,0.08)', border: '1px solid rgba(0,212,180,0.2)', borderRadius: '8px', color: 'var(--accent)', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', cursor: 'pointer', transition: 'all 0.15s' }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(0,212,180,0.14)' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(0,212,180,0.08)' }}
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v10M2 7h10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/></svg>
          New Chat
        </button>
        <div style={{ position: 'relative' }}>
          <svg width="11" height="11" viewBox="0 0 14 14" fill="none" style={{ position: 'absolute', left: '8px', top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none', color: 'var(--text-dim)' }}>
            <circle cx="6" cy="6" r="4" stroke="currentColor" strokeWidth="1.4"/>
            <path d="M9.5 9.5L12 12" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
          </svg>
          <input
            type="text"
            placeholder="Search chats..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            style={{ width: '100%', boxSizing: 'border-box', padding: '6px 8px 6px 26px', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '6px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', outline: 'none' }}
          />
        </div>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '4px 8px 12px' }}>
        {conversations.length === 0 && (
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textAlign: 'center', paddingTop: '24px' }}>No chats yet</div>
        )}
        {conversations.length > 0 && filteredConversations.length === 0 && (
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textAlign: 'center', paddingTop: '24px' }}>No results</div>
        )}
        {filteredConversations.map(c => {
          const active = c.id === activeId
          const hovered = hoveredId === c.id
          return (
            <div
              key={c.id}
              onMouseEnter={() => setHoveredId(c.id)}
              onMouseLeave={() => setHoveredId(null)}
              style={{ position: 'relative', borderRadius: '8px', marginBottom: '2px', background: active ? 'rgba(0,212,180,0.08)' : hovered ? 'rgba(255,255,255,0.04)' : 'transparent', borderLeft: active ? '2px solid var(--accent)' : '2px solid transparent', transition: 'all 0.12s' }}
            >
              <button
                onClick={() => onSelect(c.id)}
                style={{ width: '100%', textAlign: 'left', background: 'none', border: 'none', padding: '8px 28px 8px 10px', cursor: 'pointer', display: 'block' }}
              >
                <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: active ? 'var(--accent)' : 'var(--text)', fontWeight: active ? '500' : '400', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', lineHeight: 1.4 }}>
                  {c.title || 'Untitled'}
                </div>
                <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', marginTop: '2px' }}>{c.updatedAt}</div>
              </button>
              {hovered && (
                <button
                  onClick={async (e) => {
                    e.stopPropagation()
                    setDeletingId(c.id)
                    await onDelete(c.id)
                    setDeletingId(null)
                  }}
                  disabled={deletingId === c.id}
                  title="Delete"
                  style={{ position: 'absolute', right: '6px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', padding: '4px', cursor: 'pointer', color: 'var(--text-dim)', opacity: deletingId === c.id ? 0.4 : 1, display: 'flex', alignItems: 'center' }}
                  onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
                >
                  <svg width="12" height="12" viewBox="0 0 14 14" fill="none">
                    <path d="M2 4h10M5 4V2.5h4V4M5.5 6.5v4M8.5 6.5v4M3 4l.7 7.5A1 1 0 004.7 12.5h4.6a1 1 0 001-.95L11 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                </button>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function SourcesPanel({
  indexedSources,
  activeSources,
  canCustomizeSources,
  onToggle,
  citations,
  selectedSourceID,
  isSyncingSelectedSource,
  onSyncSource,
  visibleSourceSyncNotice,
}: {
  indexedSources: AppRecord[]
  activeSources: string[]
  canCustomizeSources: boolean
  onToggle: (id: string) => void
  citations: MsgSource[]
  selectedSourceID?: string
  isSyncingSelectedSource?: boolean
  onSyncSource?: () => void
  visibleSourceSyncNotice?: { kind: 'info' | 'error'; text: string } | null
}) {
  const citationSummary = citationCounts(citations)
  return (
    <div style={{ width: '260px', minWidth: '260px', height: '100%', borderLeft: '1px solid var(--border)', display: 'flex', flexDirection: 'column', background: 'var(--sidebar-bg)', overflow: 'hidden' }}>

      {/* Files section */}
      <div style={{ flexShrink: 0, padding: '16px 14px 10px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '10px' }}>
          <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', letterSpacing: '0.1em' }}>
            RECORDS · {indexedSources.length}
          </span>
          {selectedSourceID && onSyncSource && (
            <button
              onClick={onSyncSource}
              disabled={isSyncingSelectedSource}
              style={{ background: 'none', border: 'none', cursor: isSyncingSelectedSource ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', padding: '2px 4px', opacity: isSyncingSelectedSource ? 0.5 : 1 }}
            >
              {isSyncingSelectedSource ? 'Syncing…' : 'Sync'}
            </button>
          )}
        </div>
        {visibleSourceSyncNotice && (
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: visibleSourceSyncNotice.kind === 'error' ? '#ff8080' : 'var(--accent)', marginBottom: '8px', lineHeight: 1.5 }}>
            {visibleSourceSyncNotice.text}
          </div>
        )}
      </div>

      <div style={{ flex: citations.length > 0 ? '0 0 auto' : '1', maxHeight: citations.length > 0 ? '45%' : undefined, overflowY: 'auto', padding: '0 8px 8px' }}>
        {indexedSources.length === 0 && (
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textAlign: 'center', paddingTop: '12px' }}>No indexed files</div>
        )}
        {indexedSources.map(s => {
          const active = activeSources.includes(s.id)
          return (
            <button
              key={s.id}
              onClick={() => canCustomizeSources && onToggle(s.id)}
              title={s.name}
              style={{ boxSizing: 'border-box', width: '100%', display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 8px', background: active ? 'rgba(0,212,180,0.07)' : 'transparent', border: 'none', borderRadius: '6px', cursor: canCustomizeSources ? 'pointer' : 'default', textAlign: 'left', transition: 'background 0.12s' }}
              onMouseEnter={e => { if (canCustomizeSources) (e.currentTarget as HTMLButtonElement).style.background = active ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.04)' }}
              onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = active ? 'rgba(0,212,180,0.07)' : 'transparent' }}
            >
              <div style={{ width: '7px', height: '7px', borderRadius: '50%', flexShrink: 0, background: active ? 'var(--accent)' : 'rgba(255,255,255,0.15)', border: active ? 'none' : '1px solid rgba(255,255,255,0.2)', transition: 'background 0.12s' }} />
              <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: active ? 'var(--text)' : 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1, minWidth: 0 }}>
                {s.name}
              </span>
            </button>
          )
        })}
      </div>

      {/* Citations section */}
      {citations.length > 0 && (
        <>
          <div style={{ height: '1px', background: 'var(--border)', flexShrink: 0 }} />
          <div style={{ flexShrink: 0, padding: '12px 14px 8px' }}>
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', letterSpacing: '0.1em' }}>
              RECORDS · {citationSummary.records} · CHUNKS · {citationSummary.chunks}
            </span>
          </div>
          <div style={{ flex: 1, overflowY: 'auto', padding: '0 8px 12px' }}>
            {citations.map(s => (
              <div key={s.id} style={{ marginBottom: '8px', background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 10px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '5px', marginBottom: '5px' }}>
                  <svg width="10" height="10" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0, opacity: 0.5 }}>
                    <rect x="1" y="1" width="9" height="12" rx="1.5" stroke="var(--accent)" strokeWidth="1.3"/>
                    <path d="M4 5h5M4 8h3" stroke="var(--accent)" strokeWidth="1.1" strokeLinecap="round"/>
                  </svg>
                  <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--accent)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                    {s.url
                      ? <a href={s.url} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent)', textDecoration: 'none' }}>{s.doc}</a>
                      : s.doc
                    }
                  </span>
                  <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', flexShrink: 0 }}>p.{s.page}</span>
                </div>
                <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', color: 'var(--text-muted)', lineHeight: 1.6, fontStyle: 'italic' }}>
                  "{s.excerpt}"
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

export default function ChatPage() {
  const { records, chatMessages, setChatMessages, clearChatMessages, conversationId, setConversationId, conversations, setConversations, activeDomain } = useOutletContext<AppContext>()
  const { tokens } = useAuth()
  const location = useLocation()
  const accessToken = tokens?.accessToken
  const domainID = activeDomain?.id ?? ''
  const routeState = (location.state ?? null) as ChatRouteState | null
  const selectedRecord = routeState?.source
  const selectedSourceID = routeState?.sourceID
  const selectedSourceName = routeState?.sourceName
  const selectedRecordNotIndexed = !!selectedRecord && (selectedRecord.status !== 'indexed' || (selectedRecord.chunks ?? 0) <= 0)
  const [sourceScopeState, setSourceScopeState] = useState<{
    sourceID: string
    recordIDs: string[]
    error: string
  } | null>(null)
  const [isSyncingSelectedSource, setIsSyncingSelectedSource] = useState(false)
  const [sourceSyncNotice, setSourceSyncNotice] = useState<{ sourceID: string; kind: 'info' | 'error'; text: string } | null>(null)

  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingConv, setLoadingConv] = useState(false)
  const [retrievalWarning, setRetrievalWarning] = useState<string | null>(null)
  const [manualActiveSources, setManualActiveSources] = useState<string[] | null>(null)
  const [showPanel, setShowPanel] = useState(true)
  const [panelCitations, setPanelCitations] = useState<MsgSource[]>([])
  const [modelConfig] = useState(loadModelConfig)
  const [modelStatus, setModelStatus] = useState<ModelStatus>(() => initialModelStatus(loadModelConfig()))
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (modelConfig.provider !== 'local' || !toBackendModelConfig(modelConfig)) return

    let active = true
    listOllamaModels(accessToken ?? '', domainID)
      .then(models => {
        if (active) setModelStatus(models.includes(modelConfig.model) ? 'connected' : 'model-unavailable')
      })
      .catch(() => {
        if (active) setModelStatus('provider-unavailable')
      })
    return () => { active = false }
  }, [accessToken, domainID, modelConfig])

  // Auto-update panel citations when the last assistant message gets sources.
  useEffect(() => {
    for (let i = chatMessages.length - 1; i >= 0; i--) {
      if (chatMessages[i].role === 'assistant' && (chatMessages[i].sources?.length ?? 0) > 0) {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setPanelCitations(chatMessages[i].sources!)
        return
      }
    }
  }, [chatMessages])

  const handleSelectConversation = useCallback(async (id: string) => {
    if (!accessToken || loadingConv) return
    setLoadingConv(true)
    try {
      const { messages } = await getConversation(accessToken, domainID, id)
      setChatMessages(messages)
      setConversationId(id)
    } catch (err) {
      console.error('failed to load conversation:', err)
    } finally {
      setLoadingConv(false)
    }
  }, [accessToken, domainID, loadingConv, setChatMessages, setConversationId])

  const handleDeleteConversation = useCallback(async (id: string) => {
    if (!accessToken) return
    try {
      await deleteConversation(accessToken, domainID, id)
      setConversations(prev => prev.filter(c => c.id !== id))
      if (conversationId === id) {
        clearChatMessages()
      }
    } catch (err) {
      console.error('failed to delete conversation:', err)
    }
  }, [accessToken, domainID, conversationId, clearChatMessages, setConversations])

  const scopedSourceState = selectedSourceID && sourceScopeState?.sourceID === selectedSourceID
    ? sourceScopeState
    : null
  const sourceScopeError = scopedSourceState?.error ?? ''
  const isLoadingSourceScope = !!selectedSourceID && !!accessToken && !scopedSourceState
  const selectedSourceRecordIDs = useMemo(() => {
    if (!selectedSourceID) return []
    if (scopedSourceState) return scopedSourceState.recordIDs
    return records
      .filter(r => r.sourceID === selectedSourceID && r.status === 'indexed' && (r.chunks ?? 0) > 0)
      .map(r => r.id)
  }, [records, scopedSourceState, selectedSourceID])
  const selectedSourceHasNoIndexedRecords = !!selectedSourceID && !isLoadingSourceScope && selectedSourceRecordIDs.length === 0
  const allIndexedSourceIDs = useMemo(
    () => records.filter(s => s.status === 'indexed' && (s.chunks ?? 0) > 0).map(s => s.id),
    [records],
  )
  const activeSources = useMemo(() => {
    if (selectedRecord) {
      if (selectedRecordNotIndexed) return []
      return [selectedRecord.id]
    }
    if (selectedSourceID) return selectedSourceRecordIDs
    if (!manualActiveSources) return allIndexedSourceIDs
    return manualActiveSources.filter(id => allIndexedSourceIDs.includes(id))
  }, [allIndexedSourceIDs, manualActiveSources, selectedRecord, selectedRecordNotIndexed, selectedSourceID, selectedSourceRecordIDs])
  const visibleSourceSyncNotice = sourceSyncNotice && selectedSourceID === sourceSyncNotice.sourceID
    ? sourceSyncNotice
    : null
  const canCustomizeSources = !selectedRecord && !selectedSourceID

  useEffect(() => {
    if (!selectedSourceID || !accessToken) return
    let cancelled = false

    listRecordsBySource(accessToken, domainID, selectedSourceID)
      .then(nextRecords => {
        if (cancelled) return
        const indexedRecordIDs = nextRecords
          .filter(r => r.status === 'indexed' && (r.chunks ?? 0) > 0)
          .map(r => r.id)
        setSourceScopeState({ sourceID: selectedSourceID, recordIDs: indexedRecordIDs, error: '' })
      })
      .catch(err => {
        if (cancelled) return
        setSourceScopeState({
          sourceID: selectedSourceID,
          recordIDs: [],
          error: err instanceof Error ? err.message : 'Failed to load source records',
        })
      })

    return () => {
      cancelled = true
    }
  }, [selectedSourceID, accessToken, domainID])

  async function handleSyncSelectedSource() {
    if (!selectedSourceID || !accessToken || isSyncingSelectedSource) return

    setIsSyncingSelectedSource(true)
    setSourceSyncNotice({ sourceID: selectedSourceID, kind: 'info', text: 'Sync in progress...' })

    try {
      const res = await syncSource(accessToken, domainID, selectedSourceID)
      const nextRecords = await listRecordsBySource(accessToken, domainID, selectedSourceID)
      const indexedRecordIDs = nextRecords
        .filter(r => r.status === 'indexed' && (r.chunks ?? 0) > 0)
        .map(r => r.id)
      setSourceScopeState({ sourceID: selectedSourceID, recordIDs: indexedRecordIDs, error: '' })
      setSourceSyncNotice({
        sourceID: selectedSourceID,
        kind: 'info',
        text: `Sync complete: discovered ${res.discovered}, queued ${res.queued}, updated ${res.updated}, unchanged ${res.unchanged}.`,
      })
    } catch (err) {
      setSourceSyncNotice({
        sourceID: selectedSourceID,
        kind: 'error',
        text: err instanceof Error ? err.message : 'Failed to sync source',
      })
    } finally {
      setIsSyncingSelectedSource(false)
    }
  }

  useEffect(() => {
    if (bottomRef.current) bottomRef.current.scrollIntoView()
  }, [chatMessages, loading])

  const handleSend = useCallback(() => {
    if (!input.trim() || loading || !accessToken) return

    if (selectedRecordNotIndexed) {
      const explanation = selectedRecord?.error
        ? formatInlineError(selectedRecord.error)
        : '"Selected record is not indexed yet. Please retry ingest first."'
      setChatMessages(prev => [
        ...prev,
        { id: Date.now(), role: 'user', content: input.trim() },
        { id: Date.now() + 1, role: 'assistant', content: `Cannot answer for this record yet: ${explanation}` },
      ])
      setInput('')
      return
    }
    if (activeSources.length === 0) {
      const reason = selectedSourceID
        ? `"Selected source has no indexed records yet. Go to Records and wait for indexing."`
        : `"No active indexed records selected."`
      setChatMessages(prev => [
        ...prev,
        { id: Date.now(), role: 'user', content: input.trim() },
        { id: Date.now() + 1, role: 'assistant', content: `Cannot answer yet: ${reason}` },
      ])
      setInput('')
      return
    }

    const userContent = input.trim()
    const targetRecordIDs = activeSources
    setInput('')
    setLoading(true)
    setRetrievalWarning(null)

    const history: ChatMessage[] = [...chatMessages, { id: Date.now(), role: 'user', content: userContent }]
    setChatMessages(history)

    const assistantId = Date.now() + 1
    setChatMessages(prev => [...prev, { id: assistantId, role: 'assistant', content: '' }])

    const controller = new AbortController()
    abortRef.current = controller

    const apiMessages = history.map(m => ({ role: m.role, content: m.content }))

    streamChat(
      accessToken,
      domainID,
      apiMessages,
      targetRecordIDs,
      (event) => {
        if (event.type === 'warning' && event.error) {
          setRetrievalWarning(event.error)
        } else if (event.type === 'conversation' && event.conversation_id) {
          const newId = event.conversation_id
          setConversationId(newId)
          setConversations(prev => {
            if (prev.some(c => c.id === newId)) return prev
            const now = new Date().toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
            return [{ id: newId, title: userContent.slice(0, 80) || 'Untitled', createdAt: now, updatedAt: now }, ...prev]
          })
        } else if (event.type === 'token' && event.content) {
          setChatMessages(prev => prev.map(m =>
            m.id === assistantId ? { ...m, content: m.content + event.content } : m
          ))
        } else if (event.type === 'citations' && event.citations) {
          const sources = event.citations.map((c) => citationToSource(c))
          setChatMessages(prev => prev.map(m =>
            m.id === assistantId ? { ...m, sources } : m
          ))
        } else if (event.type === 'done') {
          setLoading(false)
        } else if (event.type === 'error') {
          setChatMessages(prev => prev.map(m =>
            m.id === assistantId
              ? { ...m, content: m.content || `Error: ${formatInlineError(event.error ?? 'unknown error')}` }
              : m
          ))
          setLoading(false)
        }
      },
      controller.signal,
      conversationId,
      toBackendModelConfig(loadModelConfig()),
    ).catch((err) => {
      if (err.name !== 'AbortError') {
        setChatMessages(prev => prev.map(m =>
          m.id === assistantId
            ? { ...m, content: m.content || `Error: ${formatInlineError(err instanceof Error ? err.message : 'connection error')}` }
            : m
        ))
      }
      setLoading(false)
    })
  }, [input, loading, chatMessages, setChatMessages, accessToken, domainID, activeSources, selectedRecord, selectedRecordNotIndexed, selectedSourceID, conversationId, setConversationId, setConversations])

  const indexedSources = records.filter(s => s.status === 'indexed')
  const modelStatusInfo = modelStatusDetails(modelConfig, modelStatus)

  function handleToggleSource(id: string) {
    setManualActiveSources(prev => {
      const base = prev ?? allIndexedSourceIDs
      return activeSources.includes(id) ? base.filter(x => x !== id) : [...base, id]
    })
  }

  return (
    <div style={{ flex: 1, display: 'flex', overflow: 'hidden', position: 'relative' }}>
      <ConversationList
        conversations={conversations}
        activeId={conversationId}
        onSelect={id => { void handleSelectConversation(id) }}
        onNew={clearChatMessages}
        onDelete={id => handleDeleteConversation(id)}
      />
      {loadingConv && (
        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(7,12,22,0.6)', zIndex: 20 }}>
          <div style={{ width: '24px', height: '24px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        </div>
      )}

      {/* Main chat column */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>

        {/* Header */}
        <div style={{ padding: '18px 24px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0, gap: '16px' }}>
          <div>
            <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '22px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>Prompt</h1>
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
              {activeSources.length} source{activeSources.length !== 1 ? 's' : ''} active
              {selectedSourceID && <span style={{ marginLeft: '8px', color: 'var(--text-dim)' }}>· scoped to {selectedSourceName ?? selectedSourceID}</span>}
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexShrink: 0 }}>
            <div
              title={[modelStatusInfo.provider, modelStatusInfo.model, modelStatusInfo.label].filter(Boolean).join(' · ')}
              style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '5px 9px', border: '1px solid var(--border)', borderRadius: '6px', fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', maxWidth: '240px' }}
            >
              <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: modelStatusInfo.color, flexShrink: 0 }} />
              <span style={{ color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {modelStatusInfo.provider}{modelStatusInfo.model ? ` · ${modelStatusInfo.model}` : ''}
              </span>
              <span style={{ color: modelStatusInfo.color, flexShrink: 0 }}>{modelStatusInfo.label}</span>
            </div>
            {chatMessages.length > 0 && (
              <button
                onClick={clearChatMessages}
                disabled={loading}
                title="Clear conversation"
                style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: 'none', border: '1px solid var(--border)', borderRadius: '6px', cursor: loading ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', opacity: loading ? 0.4 : 1, transition: 'all 0.15s', flexShrink: 0 }}
                onMouseEnter={e => { if (!loading) { (e.currentTarget as HTMLButtonElement).style.borderColor = '#ff5050'; (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' } }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--border)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
              >
                <svg width="11" height="11" viewBox="0 0 14 14" fill="none">
                  <path d="M2 4h10M5 4V2.5h4V4M5.5 6.5v4M8.5 6.5v4M3 4l.7 7.5A1 1 0 004.7 12.5h4.6a1 1 0 001-.95L11 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
                Clear
              </button>
            )}
            <div style={{ width: '1px', height: '20px', background: 'var(--border)', flexShrink: 0 }} />
            {/* Records panel toggle */}
            <button
              onClick={() => setShowPanel(v => !v)}
              title={showPanel ? 'Hide records panel' : 'Show records panel'}
              style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 10px', background: showPanel ? 'rgba(0,212,180,0.1)' : 'none', border: `1px solid ${showPanel ? 'rgba(0,212,180,0.35)' : 'var(--border)'}`, borderRadius: '6px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: showPanel ? 'var(--accent)' : 'var(--text-dim)', transition: 'all 0.15s' }}
            >
              <svg width="12" height="12" viewBox="0 0 14 14" fill="none">
                <rect x="1" y="1" width="9" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.3"/>
                <path d="M4 4h4M4 7h4M4 10h2" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round"/>
                <path d="M12 4v6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" opacity="0.5"/>
              </svg>
              Records
            </button>
            <div style={{ width: '1px', height: '20px', background: 'var(--border)', flexShrink: 0 }} />
            <UserMenu />
          </div>
        </div>

        {/* Error banners */}
        {selectedRecordNotIndexed && (
          <div style={{ margin: '12px 24px 0', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff8080', whiteSpace: 'nowrap', overflowX: 'auto', overflowY: 'hidden', background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.22)', borderRadius: '8px', padding: '7px 10px' }}>
            {selectedRecord?.error ? formatInlineError(selectedRecord.error) : '"Selected record is not indexed yet. Retry ingest on Records page."'}
          </div>
        )}
        {!selectedRecord && selectedSourceHasNoIndexedRecords && (
          <div style={{ margin: '12px 24px 0', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff8080', whiteSpace: 'nowrap', overflowX: 'auto', overflowY: 'hidden', background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.22)', borderRadius: '8px', padding: '7px 10px' }}>
            "Selected source has no indexed records yet. Retry sync or wait for indexing."
          </div>
        )}
        {sourceScopeError && (
          <div style={{ margin: '12px 24px 0', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff8080', whiteSpace: 'nowrap', overflowX: 'auto', overflowY: 'hidden', background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.22)', borderRadius: '8px', padding: '7px 10px' }}>
            {formatInlineError(sourceScopeError)}
          </div>
        )}

        {/* Retrieval warning */}
        {retrievalWarning && (
          <div style={{ margin: '10px 24px 0', padding: '8px 12px', background: 'rgba(255,180,0,0.07)', border: '1px solid rgba(255,180,0,0.25)', borderRadius: '8px', display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
            <svg width="13" height="13" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0, marginTop: '1px' }}>
              <path d="M7 1L13 12H1L7 1Z" stroke="#ffb400" strokeWidth="1.3" strokeLinejoin="round"/>
              <path d="M7 5.5v3M7 10h.01" stroke="#ffb400" strokeWidth="1.3" strokeLinecap="round"/>
            </svg>
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ffb400', lineHeight: 1.5 }}>{retrievalWarning}</span>
            <button onClick={() => setRetrievalWarning(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#ffb400', opacity: 0.6, padding: '0 0 0 4px', marginLeft: 'auto', flexShrink: 0, fontSize: '14px', lineHeight: 1 }}>×</button>
          </div>
        )}

        {/* Messages */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '0 32px' }}>
          <div style={{ maxWidth: '760px', margin: '0 auto', width: '100%', padding: '32px 0' }}>
            {chatMessages.length === 0 && !loading && (
              <div style={{ textAlign: 'center', paddingTop: '80px', color: 'var(--text-dim)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px' }}>
                Ask a question about your indexed records.
              </div>
            )}
            {chatMessages.map(msg => (
              <MessageBubble
                key={msg.id}
                msg={msg}
                onShowSources={sources => {
                  setPanelCitations(sources)
                  setShowPanel(true)
                }}
              />
            ))}
            {loading && !chatMessages[chatMessages.length - 1]?.content && <TypingIndicator />}
            <div ref={bottomRef} />
          </div>
        </div>

        {/* Input */}
        <div style={{ padding: '16px 32px 24px', borderTop: '1px solid var(--border)', flexShrink: 0 }}>
          <div style={{ maxWidth: '760px', margin: '0 auto', width: '100%' }}>
            <div style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '12px', overflow: 'hidden' }}>
              <textarea
                value={input}
                onChange={e => setInput(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }}
                placeholder="Ask a question about your documents…"
                rows={1}
                style={{ width: '100%', background: 'transparent', border: 'none', outline: 'none', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', padding: '14px 16px 10px', resize: 'none', lineHeight: 1.6, boxSizing: 'border-box', minHeight: '48px' }}
                onInput={e => { const t = e.target as HTMLTextAreaElement; t.style.height = 'auto'; t.style.height = Math.min(t.scrollHeight, 160) + 'px' }}
              />
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '8px 12px 10px' }}>
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>RAG · Grounded responses only</span>
                {loading ? (
                  <button
                    onClick={() => { abortRef.current?.abort(); setLoading(false) }}
                    style={{ background: 'rgba(255,80,80,0.12)', color: '#ff5050', border: '1px solid rgba(255,80,80,0.3)', padding: '7px 14px', borderRadius: '7px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', transition: 'all 0.15s' }}
                    onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,80,80,0.22)' }}
                    onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,80,80,0.12)' }}
                  >
                    <svg width="12" height="12" viewBox="0 0 12 12" fill="none"><rect x="2" y="2" width="8" height="8" rx="1.5" fill="currentColor"/></svg>
                    Stop
                  </button>
                ) : (
                  <button onClick={handleSend} disabled={!input.trim()}
                    style={{ background: input.trim() ? 'var(--accent)' : 'var(--border)', color: input.trim() ? '#070c16' : 'var(--text-dim)', border: 'none', padding: '7px 14px', borderRadius: '7px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '6px', cursor: input.trim() ? 'pointer' : 'default', transition: 'all 0.15s' }}
                  >
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2 7h10M8 3l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
                    Send
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Right sources panel */}
      {showPanel && (
        <SourcesPanel
          indexedSources={indexedSources}
          activeSources={activeSources}
          canCustomizeSources={canCustomizeSources}
          onToggle={handleToggleSource}
          citations={panelCitations}
          selectedSourceID={selectedSourceID}
          isSyncingSelectedSource={isSyncingSelectedSource}
          onSyncSource={() => { void handleSyncSelectedSource() }}
          visibleSourceSyncNotice={visibleSourceSyncNotice}
        />
      )}
    </div>
  )
}
