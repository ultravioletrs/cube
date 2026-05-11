import { useCallback, useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from '@/components/Sidebar'
import { useAuth } from '@/hooks/useAuth'
import { listConversations, listRecords, listSources } from '@/lib/embedder/service'
import type { AppContext, AppRecord, ChatMessage, Conversation, DriveSource } from '@/types'

const CHAT_KEY = 'veda_chat'
const CONV_KEY = 'veda_conv_id'

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
  const persistTimer = useRef<number | null>(null)

  const clearChatMessages = useCallback(() => {
    setChatMessages([])
    setConversationId(null)
    localStorage.removeItem(CHAT_KEY)
    localStorage.removeItem(CONV_KEY)
  }, [])

  useEffect(() => {
    if (!tokens?.accessToken) return
    const token = tokens.accessToken
    Promise.all([listRecords(token), listSources(token)])
      .then(([recs, srcs]) => { setRecords(recs); setDriveSources(srcs) })
      .catch((err: unknown) => console.error('failed to load records/sources:', err))
    listConversations(token)
      .then(convs => setConversations(convs))
      .catch((err: unknown) => console.error('failed to load conversations:', err))
  }, [tokens?.accessToken])

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

  const context: AppContext = { records, setRecords, driveSources, setDriveSources, chatMessages, setChatMessages, clearChatMessages, conversationId, setConversationId, conversations, setConversations }

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden', background: 'var(--bg)' }}>
      <Sidebar />
      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Outlet context={context} />
      </main>
    </div>
  )
}
