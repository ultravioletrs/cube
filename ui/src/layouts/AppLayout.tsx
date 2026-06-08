// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from '@/components/Sidebar'
import { useAuth } from '@/hooks/useAuth'
import { listConversations, listRecords, listSources } from '@/lib/embedder/service'
import type { ActiveWorkspace, AppContext, AppRecord, ChatMessage, Conversation, DriveSource } from '@/types'

const CHAT_KEY = 'cube_chat'
const CONV_KEY = 'cube_conv_id'
const TENANT_KEY = 'cube_active_tenant'
const LEGACY_DOMAIN_KEY = 'cube_active_domain'

function loadChatMessages(): ChatMessage[] {
  try {
    const raw = localStorage.getItem(CHAT_KEY)
    if (!raw) return []
    return JSON.parse(raw) as ChatMessage[]
  } catch {
    return []
  }
}

export default function AppLayout() {
  const { tokens } = useAuth()
  const [records, setRecords] = useState<AppRecord[]>([])
  const [driveSources, setDriveSources] = useState<DriveSource[]>([])
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>(loadChatMessages)
  const [conversationId, setConversationId] = useState<string | null>(() => localStorage.getItem(CONV_KEY))
  const [activeWorkspace, setActiveWorkspace] = useState<ActiveWorkspace | null>(() => {
    try {
      const raw = localStorage.getItem(TENANT_KEY) ?? localStorage.getItem(LEGACY_DOMAIN_KEY)
      return raw ? (JSON.parse(raw) as ActiveWorkspace) : null

    } catch {
      return null
    }
  })
  const persistTimer = useRef<number | null>(null)

  const clearChatMessages = useCallback(() => {
    setChatMessages([])
    setConversationId(null)
    localStorage.removeItem(CHAT_KEY)
    localStorage.removeItem(CONV_KEY)
  }, [])

  const workspaceID = activeWorkspace?.id ?? ''

  useEffect(() => {
    // Clear stale data from the previous workspace immediately so pages never
    // briefly show records that belong to a different workspace.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setRecords([])
    setDriveSources([])
    setConversations([])

    if (!tokens?.accessToken || !workspaceID) return

    let cancelled = false
    const token = tokens.accessToken

    Promise.all([listRecords(token, workspaceID), listSources(token, workspaceID)])
      .then(([recs, srcs]) => {
        if (cancelled) return
        setRecords(recs)
        setDriveSources(srcs)
      })
      .catch((err: unknown) => console.error('failed to load records/sources:', err))

    listConversations(token, workspaceID)
      .then(convs => { if (!cancelled) setConversations(convs) })
      .catch((err: unknown) => console.error('failed to load conversations:', err))

    return () => { cancelled = true }
  }, [tokens?.accessToken, workspaceID])

  useEffect(() => {
    if (persistTimer.current !== null) clearTimeout(persistTimer.current)
    persistTimer.current = window.setTimeout(() => {
      localStorage.setItem(CHAT_KEY, JSON.stringify(chatMessages.slice(-50)))
    }, 500)
    return () => {
      if (persistTimer.current !== null) clearTimeout(persistTimer.current)
    }
  }, [chatMessages])

  useEffect(() => {
    if (conversationId) {
      localStorage.setItem(CONV_KEY, conversationId)
    } else {
      localStorage.removeItem(CONV_KEY)
    }
  }, [conversationId])

  useEffect(() => {
    if (activeWorkspace) {
      localStorage.setItem(TENANT_KEY, JSON.stringify(activeWorkspace))
      localStorage.removeItem(LEGACY_DOMAIN_KEY)
    } else {
      localStorage.removeItem(TENANT_KEY)
      localStorage.removeItem(LEGACY_DOMAIN_KEY)
    }
  }, [activeWorkspace])

  const context: AppContext = {
    records,
    setRecords,
    driveSources,
    setDriveSources,
    chatMessages,
    setChatMessages,
    clearChatMessages,
    conversationId,
    setConversationId,
    conversations,
    setConversations,
    activeWorkspace,
    setActiveWorkspace,
    activeDomain: activeWorkspace,
    setActiveDomain: setActiveWorkspace,
  }

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden', background: 'var(--bg)' }}>
      <Sidebar activeWorkspace={activeWorkspace} />
      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Outlet context={context} />
      </main>
    </div>
  )
}
