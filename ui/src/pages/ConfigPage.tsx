// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useState, useEffect } from 'react'
import { useOutletContext } from 'react-router-dom'
import UserMenu from '@/components/UserMenu'
import { useAuth } from '@/hooks/useAuth'
import { loadModelConfig, saveModelConfig, DEFAULT_MODEL_CONFIG } from '@/lib/modelConfig'
import type { LLMProvider } from '@/lib/modelConfig'
import { getGuardrailsStatus, listOllamaModels, setGuardrailsEnabled } from '@/lib/api'
import type { AppContext } from '@/types'

function Toggle({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div onClick={() => onChange(!value)} style={{ width: '40px', height: '22px', borderRadius: '11px', background: value ? 'var(--accent)' : 'var(--border)', cursor: 'pointer', position: 'relative', transition: 'background 0.2s', flexShrink: 0 }}>
      <div style={{ position: 'absolute', top: '3px', left: value ? '21px' : '3px', width: '16px', height: '16px', borderRadius: '50%', background: value ? '#070c16' : 'var(--text-dim)', transition: 'left 0.2s' }} />
    </div>
  )
}

function Slider({ value, onChange, min, max, step, label }: { value: number; onChange: (v: number) => void; min: number; max: number; step: number; label: string }) {
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)' }}>{label}</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color: 'var(--accent)' }}>{value}</span>
      </div>
      <input type="range" min={min} max={max} step={step} value={value} onChange={e => onChange(Number(e.target.value))} style={{ width: '100%', accentColor: 'var(--accent)', cursor: 'pointer' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '4px' }}>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>{min}</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>{max}</span>
      </div>
    </div>
  )
}

function SelectInput({ value, onChange, options }: { value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }) {
  return (
    <select value={value} onChange={e => onChange(e.target.value)} style={{ background: '#111827', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', width: '100%', outline: 'none', cursor: 'pointer' }}>
      {options.map(o => <option key={o.value} value={o.value} style={{ background: '#111827', color: '#e2e8f0' }}>{o.label}</option>)}
    </select>
  )
}

function Section({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <div style={{ paddingTop: '32px', paddingBottom: '8px', display: 'flex', flexDirection: 'column', gap: '0' }}>
      <div style={{ marginBottom: '20px' }}>
        <h2 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '16px', color: 'var(--text)', margin: '0 0 4px', letterSpacing: '-0.01em' }}>{title}</h2>
        {subtitle && <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12.5px', color: 'var(--text-muted)', margin: 0, lineHeight: 1.5 }}>{subtitle}</p>}
      </div>
      {children}
    </div>
  )
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: '20px' }}>
      <div style={{ marginBottom: '8px' }}>
        <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)', marginBottom: '3px' }}>{label}</div>
        {hint && <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-dim)', marginBottom: '8px', lineHeight: 1.5 }}>{hint}</div>}
      </div>
      {children}
    </div>
  )
}

function ProviderButtons({ providers, active, onSelect, labelMap }: { providers: string[]; active: string; onSelect: (p: string) => void; labelMap: Record<string, string> }) {
  return (
    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
      {providers.map(p => (
        <button key={p} onClick={() => onSelect(p)} style={{ background: active === p ? 'rgba(0,212,180,0.15)' : 'transparent', border: `1px solid ${active === p ? 'rgba(0,212,180,0.5)' : 'var(--border)'}`, color: active === p ? 'var(--accent)' : 'var(--text-muted)', borderRadius: '8px', padding: '7px 14px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '500', cursor: 'pointer', transition: 'all 0.15s' }}>
          {labelMap[p] ?? p}
        </button>
      ))}
    </div>
  )
}

const providerLabel: Record<string, string> = { openai: 'OpenAI', anthropic: 'Anthropic', local: 'Local / Ollama', cohere: 'Cohere' }

const llmModels: Record<string, { value: string; label: string }[]> = {
  openai: [{ value: 'gpt-4o', label: 'GPT-4o' }, { value: 'gpt-4o-mini', label: 'GPT-4o Mini' }, { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' }, { value: 'o3', label: 'o3 (reasoning)' }],
  anthropic: [{ value: 'claude-opus-4-5', label: 'Claude Opus 4.5' }, { value: 'claude-sonnet-4-5', label: 'Claude Sonnet 4.5' }, { value: 'claude-haiku-4-5', label: 'Claude Haiku 4.5' }],
}

const embModels: Record<string, { value: string; label: string }[]> = {
  openai: [{ value: 'text-embedding-3-large', label: 'text-embedding-3-large (3072d)' }, { value: 'text-embedding-3-small', label: 'text-embedding-3-small (1536d)' }],
  local: [{ value: 'bge-m3', label: 'BGE-M3 (1024d)' }, { value: 'nomic-embed', label: 'Nomic Embed (768d)' }, { value: 'all-minilm', label: 'all-MiniLM-L6 (384d)' }],
  cohere: [{ value: 'embed-v4', label: 'Cohere Embed v4 (1024d)' }, { value: 'embed-multilingual', label: 'Embed Multilingual (1024d)' }],
}

export default function ConfigPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()
  const accessToken = tokens?.accessToken ?? ''
  const domainID = activeDomain?.id ?? ''

  const [llmProvider, setLlmProvider] = useState<LLMProvider>(DEFAULT_MODEL_CONFIG.provider)
  const [llmModel, setLlmModel] = useState(DEFAULT_MODEL_CONFIG.model)
  const [apiKey, setApiKey] = useState(DEFAULT_MODEL_CONFIG.apiKey)
  const [temperature, setTemperature] = useState(DEFAULT_MODEL_CONFIG.temperature)
  const [maxTokens, setMaxTokens] = useState(DEFAULT_MODEL_CONFIG.maxTokens)
  const [streamResponses, setStreamResponses] = useState(DEFAULT_MODEL_CONFIG.streamResponses)
  const [systemPrompt, setSystemPrompt] = useState(DEFAULT_MODEL_CONFIG.systemPrompt)
  const [embProvider, setEmbProvider] = useState('openai')
  const [embModel, setEmbModel] = useState('text-embedding-3-large')
  const [chunkSize, setChunkSize] = useState(512)
  const [chunkOverlap, setChunkOverlap] = useState(64)
  const [topK, setTopK] = useState(5)
  const [saved, setSaved] = useState(false)
  const [ollamaModels, setOllamaModels] = useState<string[]>([])
  const [ollamaLoading, setOllamaLoading] = useState(false)
  const [guardrailsEnabled, setGuardrailsEnabledState] = useState(false)
  const [guardrailsConfigured, setGuardrailsConfigured] = useState(false)
  const [guardrailsLoading, setGuardrailsLoading] = useState(false)

  useEffect(() => {
    if (!tokens?.accessToken || !domainID) return
    getGuardrailsStatus(tokens.accessToken, domainID)
      .then(s => { setGuardrailsEnabledState(s.enabled); setGuardrailsConfigured(s.configured) })
      .catch(() => {})
  }, [tokens?.accessToken, domainID])

  const handleGuardrailsToggle = async (v: boolean) => {
    if (!tokens?.accessToken || !domainID || !guardrailsConfigured || guardrailsLoading) return
    setGuardrailsLoading(true)
    try {
      const s = await setGuardrailsEnabled(tokens.accessToken, v, domainID)
      setGuardrailsEnabledState(s.enabled)
    } catch { /* ignore */ } finally {
      setGuardrailsLoading(false)
    }
  }

  useEffect(() => {
    const cfg = loadModelConfig()
    setLlmProvider(cfg.provider)
    setLlmModel(cfg.model)
    setApiKey(cfg.apiKey)
    setTemperature(cfg.temperature)
    setMaxTokens(cfg.maxTokens)
    setStreamResponses(cfg.streamResponses)
    setSystemPrompt(cfg.systemPrompt)
  }, [])

  useEffect(() => {
    if (llmProvider !== 'local' || !domainID) return
    setOllamaLoading(true)
    listOllamaModels(accessToken, domainID)
      .then(models => {
        setOllamaModels(models)
        // Auto-select the first model if none is currently set
        setLlmModel(prev => prev || (models[0] ?? ''))
      })
      .catch(() => { setOllamaModels([]) })
      .finally(() => { setOllamaLoading(false) })
  }, [llmProvider, accessToken, domainID])

  const handleProviderChange = (p: string) => {
    const provider = p as LLMProvider
    setLlmProvider(provider)
    if (provider === 'local') {
      setLlmModel(ollamaModels[0] ?? '')
    } else {
      setLlmModel(llmModels[provider]?.[0]?.value ?? '')
    }
  }

  const currentModelOptions = llmProvider === 'local'
    ? ollamaModels.map(m => ({ value: m, label: m }))
    : llmModels[llmProvider] ?? []

  const handleSave = () => {
    saveModelConfig({ provider: llmProvider, model: llmModel, apiKey, temperature, maxTokens, streamResponses, systemPrompt })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div style={{ padding: '24px 32px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '22px', color: 'var(--text)', margin: 0, letterSpacing: '-0.02em' }}>Configuration</h1>
          <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', margin: '4px 0 0' }}>Model stack · per workspace</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <button onClick={handleSave} style={{ padding: '9px 20px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', display: 'flex', alignItems: 'center', gap: '7px', transition: 'all 0.2s', background: saved ? '#22c55e20' : 'var(--accent)', color: saved ? '#22c55e' : '#070c16', border: saved ? '1px solid #22c55e40' : 'none' }}>
            {saved ? (<><svg width="13" height="13" viewBox="0 0 13 13" fill="none"><path d="M2 6.5l3 3 6-6" stroke="#22c55e" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>Saved</>) : 'Save Changes'}
          </button>
          <div style={{ width: '1px', height: '20px', background: 'var(--border)' }} />
          <UserMenu />
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '0 32px' }}>
        <div style={{ maxWidth: '760px', width: '100%', margin: '0 auto', display: 'flex', flexDirection: 'column', paddingBottom: '40px' }}>
          <Section title="Language Model" subtitle="Select the LLM used to generate responses from retrieved context.">
            <Field label="Provider"><ProviderButtons providers={['openai', 'anthropic', 'local']} active={llmProvider} onSelect={handleProviderChange} labelMap={providerLabel} /></Field>
            {llmProvider !== 'local' && (
              <Field label="API Key" hint={`Your ${providerLabel[llmProvider]} API key. Stored locally in your browser only.`}>
                <input type="password" value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="sk-..." style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', outline: 'none', boxSizing: 'border-box' }} />
              </Field>
            )}
            <Field label="Model" hint={llmProvider === 'local' ? 'Models fetched live from your Ollama instance.' : 'The selected model must support function calling for optimal RAG performance.'}>
              {llmProvider === 'local' && ollamaLoading
                ? <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', padding: '9px 0' }}>Loading models from Ollama…</div>
                : <SelectInput value={llmModel} onChange={setLlmModel} options={currentModelOptions} />
              }
              {llmProvider === 'local' && !ollamaLoading && ollamaModels.length === 0 && (
                <div style={{ marginTop: '8px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ffb400' }}>No local models found — make sure Ollama is running.</div>
              )}
            </Field>
            <Field label="Temperature" hint="Lower values produce more deterministic, grounded responses."><Slider value={temperature} onChange={setTemperature} min={0} max={1} step={0.05} label="temperature" /></Field>
            <Field label="Max Output Tokens"><Slider value={maxTokens} onChange={setMaxTokens} min={256} max={4096} step={128} label="tokens" /></Field>
            <Field label="Stream responses">
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <Toggle value={streamResponses} onChange={setStreamResponses} />
                <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)' }}>{streamResponses ? 'Enabled — tokens streamed progressively' : 'Disabled — wait for full response'}</span>
              </div>
            </Field>
            <Field label="System prompt" hint="Injected before every conversation. Keep it focused on grounding.">
              <textarea value={systemPrompt} onChange={e => setSystemPrompt(e.target.value)} rows={4} style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '10px 12px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '11.5px', lineHeight: 1.6, resize: 'vertical', outline: 'none', boxSizing: 'border-box' }} />
            </Field>
          </Section>

          <div style={{ height: '1px', background: 'var(--border)', margin: '8px 0 32px' }} />

          <Section title="Embedding Model" subtitle="Controls how documents are chunked and vectorised for retrieval.">
            <Field label="Provider"><ProviderButtons providers={['openai', 'local', 'cohere']} active={embProvider} onSelect={p => { setEmbProvider(p); setEmbModel(embModels[p][0].value) }} labelMap={providerLabel} /></Field>
            <Field label="Embedding model" hint="Changing the model requires re-indexing all existing documents.">
              <SelectInput value={embModel} onChange={setEmbModel} options={embModels[embProvider] ?? []} />
              <div style={{ marginTop: '8px', padding: '8px 12px', background: 'rgba(255,180,0,0.07)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '7px', display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                <svg width="13" height="13" viewBox="0 0 13 13" fill="none" style={{ flexShrink: 0, marginTop: '1px' }}><path d="M6.5 1L12 11.5H1L6.5 1z" stroke="#ffb400" strokeWidth="1.2" strokeLinejoin="round"/><path d="M6.5 5v3M6.5 9.5v.5" stroke="#ffb400" strokeWidth="1.2" strokeLinecap="round"/></svg>
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ffb400' }}>Embeddings must be compatible with your vector database schema. Changing provider requires schema migration.</span>
              </div>
            </Field>
            <Field label="Chunk size (tokens)" hint="Larger chunks preserve context; smaller chunks improve retrieval precision."><Slider value={chunkSize} onChange={setChunkSize} min={128} max={2048} step={64} label="tokens" /></Field>
            <Field label="Chunk overlap (tokens)" hint="Overlap between adjacent chunks to avoid cutting context boundaries."><Slider value={chunkOverlap} onChange={setChunkOverlap} min={0} max={256} step={16} label="tokens" /></Field>
            <Field label="Top-K retrieval" hint="Number of chunks retrieved per query. Higher values increase recall but may dilute relevance."><Slider value={topK} onChange={setTopK} min={1} max={20} step={1} label="chunks" /></Field>
          </Section>

          <div style={{ height: '1px', background: 'var(--border)', margin: '8px 0 32px' }} />

          <Section title="Safety" subtitle="Control content guardrails applied to all chat queries before they reach the language model.">
            <Field
              label="Content guardrails"
              hint={guardrailsConfigured ? 'Filters harmful, manipulative, or sensitive content in real-time before queries are processed.' : 'Guardrails service is not configured. Set EMBEDDER_GUARDRAILS_URL to enable.'}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', opacity: guardrailsConfigured ? 1 : 0.45, pointerEvents: guardrailsConfigured ? 'auto' : 'none' }}>
                <Toggle value={guardrailsEnabled} onChange={handleGuardrailsToggle} />
                <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)' }}>
                  {guardrailsLoading ? 'Updating…' : guardrailsEnabled ? 'Enabled — harmful queries blocked before reaching the model' : 'Disabled — all queries pass through unfiltered'}
                </span>
              </div>
            </Field>
          </Section>

          <div style={{ height: '1px', background: 'var(--border)', margin: '8px 0 32px' }} />

          <div style={{ padding: '14px 16px', borderRadius: '10px', background: 'rgba(0,212,180,0.05)', border: '1px solid rgba(0,212,180,0.15)', display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ flexShrink: 0, marginTop: '1px' }}><circle cx="7" cy="7" r="6" stroke="var(--accent)" strokeWidth="1.2"/><path d="M7 5v4M7 4v.5" stroke="var(--accent)" strokeWidth="1.4" strokeLinecap="round"/></svg>
            <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12.5px', color: 'var(--text-muted)', lineHeight: 1.6 }}>All configuration changes apply to <strong style={{ color: 'var(--text)' }}>your workspace only</strong> and do not affect other users or workspaces.</span>
          </div>
        </div>
      </div>
    </div>
  )
}
