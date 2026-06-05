// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AppRecord, ChatDebug } from '@/types'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface Citation {
  record_id: string
  record_name: string
  external_url?: string
  chunk_index: number
  excerpt: string
}

export type ChatEventType = 'token' | 'citations' | 'debug' | 'error' | 'done' | 'conversation' | 'warning'

export interface ChatEvent {
  type: ChatEventType
  content?: string
  citations?: Citation[]
  debug?: ChatDebug
  error?: string
  conversation_id?: string
}

// listRecords fetches the domain's records from the embedder API.
export async function listRecords(token: string, domainID: string): Promise<AppRecord[]> {
  const headers: Record<string, string> = { Authorization: `Bearer ${token}` }
  if (domainID) headers['X-Domain-Id'] = domainID
  const res = await fetch('/api/v1/records', { headers })
  if (!res.ok) throw new Error(`listRecords: ${res.status}`)
  const data = await res.json()
  return (data.records ?? []).map(toAppRecord)
}

// BackendModelConfig is the model override shape accepted by POST /api/v1/chat.
export interface BackendModelConfig {
  provider: string
  base_url: string
  model: string
  api_key: string
  temperature: number
  max_tokens: number
}

// streamChat opens an SSE connection to the chat endpoint and calls onEvent
// for each parsed event.  Returns a cleanup function that aborts the stream.
export function streamChat(
  token: string,
  domainID: string,
  messages: ChatMessage[],
  recordIDs: string[],
  onEvent: (event: ChatEvent) => void,
  signal?: AbortSignal,
  conversationId?: string | null,
  modelConfig?: BackendModelConfig | null,
  debug?: boolean,
): Promise<void> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  }
  if (domainID) headers['X-Domain-Id'] = domainID
  return fetch('/api/v1/chat', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      messages,
      record_ids: recordIDs,
      conversation_id: conversationId ?? undefined,
      model: modelConfig ?? undefined,
      debug: debug || undefined,
    }),
    signal,
  }).then(async (res) => {
    if (!res.ok) {
      let msg = `streamChat: ${res.status}`
      try {
        const text = await res.text()
        if (text.trim()) {
          try {
            const json = JSON.parse(text) as { error?: string; message?: string }
            msg = json.error || json.message || text
          } catch {
            msg = text
          }
        }
      } catch {
        // keep default msg
      }
      throw new Error(msg)
    }
    const reader = res.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const parts = buffer.split('\n\n')
      buffer = parts.pop() ?? ''
      for (const part of parts) {
        const line = part.trim()
        if (!line.startsWith('data: ')) continue
        const json = line.slice('data: '.length)
        try {
          const event: ChatEvent = JSON.parse(json)
          onEvent(event)
        } catch {
          // skip malformed lines
        }
      }
    }
  })
}

// toAppRecord maps the backend record response to an AppRecord.
function toAppRecord(r: Record<string, unknown>): AppRecord {
  return {
    id: r['id'] as string,
    name: r['name'] as string,
    format: (r['format'] as AppRecord['format']) ?? 'pdf',
    status: mapStatus(r['status'] as string),
    createdAt: (r['created_at'] as string) ?? '',
    description: (r['description'] as string) ?? '',
    chunks: (r['chunks'] as number | null) ?? null,
    size: r['size_bytes'] != null ? formatBytes(r['size_bytes'] as number) : undefined,
    pages: (r['pages'] as number | null) ?? null,
    url: (r['external_url'] as string) || undefined,
  }
}

function mapStatus(s: string): AppRecord['status'] {
  if (s === 'queued') return 'queued'
  if (s === 'processing') return 'processing'
  if (s === 'indexed') return 'indexed'
  if (s === 'failed') return 'failed'
  if (s === 'cancelled') return 'cancelled'
  return 'processing'
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// listOllamaModels fetches the locally available Ollama models from the embedder proxy.
export async function listOllamaModels(token: string, domainID: string): Promise<string[]> {
  const headers: Record<string, string> = { Authorization: `Bearer ${token}` }
  if (domainID) headers['X-Domain-Id'] = domainID
  const res = await fetch('/api/v1/models/ollama', { headers })
  if (!res.ok) throw new Error(`listOllamaModels: ${res.status}`)
  const data = await res.json() as { models: string[] }
  return data.models ?? []
}

export interface ModelConnectionResult {
  connected: boolean
  message: string
}

export async function testModelConnection(
  token: string,
  config: BackendModelConfig,
): Promise<ModelConnectionResult> {
  const res = await fetch('/api/v1/models/test-connection', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify(config),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null) as { error?: string } | null
    throw new Error(body?.error || `model connection test: ${res.status}`)
  }
  return res.json() as Promise<ModelConnectionResult>
}

export interface GuardrailsStatus {
  enabled: boolean
  configured: boolean
}

export async function getGuardrailsStatus(token: string): Promise<GuardrailsStatus> {
  const res = await fetch('/api/v1/guardrails', {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`guardrails status: ${res.status}`)
  return res.json() as Promise<GuardrailsStatus>
}

export async function setGuardrailsEnabled(token: string, enabled: boolean): Promise<GuardrailsStatus> {
  const res = await fetch('/api/v1/guardrails', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ enabled }),
  })
  if (!res.ok) throw new Error(`guardrails update: ${res.status}`)
  return res.json() as Promise<GuardrailsStatus>
}
