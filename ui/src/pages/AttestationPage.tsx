// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { getAttestationPolicy, updateAttestationPolicy } from '@/lib/attestation'
import type { AttestationPolicy } from '@/lib/attestation'
import type { AppContext } from '@/types'

function StatusBadge({ ok, label }: { ok: boolean; label: string }) {
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '5px',
      background: ok ? 'rgba(0,212,180,0.1)' : 'rgba(255,80,80,0.1)',
      color: ok ? '#00d4b4' : '#ff5050',
      fontFamily: 'JetBrains Mono, monospace', fontSize: '10px',
      padding: '3px 10px', borderRadius: '20px', fontWeight: '500',
    }}>
      <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: 'currentColor', display: 'inline-block' }} />
      {label}
    </span>
  )
}

function InfoCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{
      background: 'rgba(0,212,180,0.04)', border: '1px solid rgba(0,212,180,0.15)',
      borderRadius: '10px', padding: '16px 20px',
    }}>
      <p style={{ margin: '0 0 8px', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: '#00d4b4', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
        {title}
      </p>
      {children}
    </div>
  )
}

export default function AttestationPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [policy, setPolicy] = useState<AttestationPolicy | null>(null)
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [parseError, setParseError] = useState<string | null>(null)

  const domainID = activeDomain?.id ?? ''

  const load = useCallback(async () => {
    if (!tokens?.accessToken || !domainID) return
    setLoading(true)
    setError(null)
    try {
      const p = await getAttestationPolicy(tokens.accessToken, domainID)
      setPolicy(p)
      setDraft(p ? JSON.stringify(p, null, 2) : '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load policy')
    } finally {
      setLoading(false)
    }
  }, [tokens?.accessToken, domainID])

  useEffect(() => { void load() }, [load])

  function handleDraftChange(value: string) {
    setDraft(value)
    setSaved(false)
    try {
      JSON.parse(value)
      setParseError(null)
    } catch {
      setParseError('Invalid JSON')
    }
  }

  async function handleSave() {
    if (!tokens?.accessToken) return
    setSaveError(null)
    setSaving(true)
    try {
      const parsed = JSON.parse(draft) as AttestationPolicy
      await updateAttestationPolicy(tokens.accessToken, parsed)
      setPolicy(parsed)
      setSaved(true)
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Failed to save policy')
    } finally {
      setSaving(false)
    }
  }

  const hasChanges = draft !== (policy ? JSON.stringify(policy, null, 2) : '')
  const canSave = !parseError && hasChanges && !saving && !!draft.trim()

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '24px 28px 0', borderBottom: '1px solid var(--border)', paddingBottom: '16px', flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <h1 style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', letterSpacing: '-0.02em' }}>
              Attestation
            </h1>
            <p style={{ margin: '4px 0 0', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>
              Manage aTLS attestation policy for domain: {domainID || '—'}
            </p>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {policy !== null ? (
              <StatusBadge ok label="Policy configured" />
            ) : (
              !loading && <StatusBadge ok={false} label="No policy" />
            )}
          </div>
        </div>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflow: 'auto', padding: '24px 28px', display: 'flex', gap: '24px' }}>
        {/* Left — editor */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '12px', minWidth: 0 }}>
          {!domainID && (
            <div style={{ background: 'rgba(255,180,0,0.08)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '10px', padding: '14px 18px' }}>
              <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ffb400' }}>
                Select a domain to view or edit its attestation policy.
              </p>
            </div>
          )}

          {error && (
            <div style={{ background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.2)', borderRadius: '10px', padding: '14px 18px' }}>
              <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color: '#ff5050' }}>{error}</p>
            </div>
          )}

          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text-muted)' }}>
              Policy (JSON)
            </p>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              {parseError && (
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff5050' }}>{parseError}</span>
              )}
              {saved && !hasChanges && (
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#00d4b4' }}>Saved</span>
              )}
              <button
                onClick={() => void load()}
                disabled={loading || !domainID}
                style={{
                  background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border)',
                  borderRadius: '7px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif',
                  fontSize: '12px', padding: '5px 12px', cursor: 'pointer', opacity: loading ? 0.5 : 1,
                }}
              >
                {loading ? 'Loading…' : 'Refresh'}
              </button>
              <button
                onClick={() => void handleSave()}
                disabled={!canSave}
                style={{
                  background: canSave ? 'var(--accent)' : 'rgba(0,212,180,0.2)',
                  border: 'none', borderRadius: '7px',
                  color: canSave ? '#070c16' : 'var(--text-dim)',
                  fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600',
                  fontSize: '12px', padding: '5px 14px', cursor: canSave ? 'pointer' : 'not-allowed',
                  transition: 'all 0.12s ease',
                }}
              >
                {saving ? 'Saving…' : 'Save Policy'}
              </button>
            </div>
          </div>

          {saveError && (
            <div style={{ background: 'rgba(255,80,80,0.08)', border: '1px solid rgba(255,80,80,0.2)', borderRadius: '8px', padding: '10px 14px' }}>
              <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: '#ff5050' }}>{saveError}</p>
            </div>
          )}

          <textarea
            value={draft}
            onChange={e => handleDraftChange(e.target.value)}
            disabled={!domainID || loading}
            placeholder={loading ? 'Loading policy…' : '{\n  "platform": "snp",\n  "measurements": {}\n}'}
            spellCheck={false}
            style={{
              flex: 1,
              minHeight: '400px',
              background: 'var(--card-bg)',
              border: `1px solid ${parseError ? 'rgba(255,80,80,0.4)' : 'var(--border)'}`,
              borderRadius: '10px',
              color: 'var(--text)',
              fontFamily: 'JetBrains Mono, monospace',
              fontSize: '12px',
              lineHeight: '1.6',
              padding: '16px',
              resize: 'vertical',
              outline: 'none',
              transition: 'border-color 0.12s ease',
            }}
          />
        </div>

        {/* Right — info panel */}
        <div style={{ width: '280px', flexShrink: 0, display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <InfoCard title="What is Attestation?">
            <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
              Attestation (aTLS) verifies that the Cube agent is running inside a trusted
              Confidential Computing environment — AMD SEV-SNP, Intel TDX, or Azure CVM —
              before any data is processed.
            </p>
          </InfoCard>

          <InfoCard title="Policy Format">
            <p style={{ margin: '0 0 8px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
              The policy is stored as JSON. Supported platforms:
            </p>
            <ul style={{ margin: 0, padding: '0 0 0 16px', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', lineHeight: 1.8 }}>
              <li>snp — AMD SEV-SNP</li>
              <li>tdx — Intel TDX</li>
              <li>snp_vtpm — SNP + vTPM</li>
              <li>azure — Azure CVM</li>
              <li>no_cc — No CC (dev/test)</li>
            </ul>
          </InfoCard>

          <InfoCard title="How It Works">
            <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
              When a request reaches the proxy, the router matches it against the
              <code style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', background: 'rgba(255,255,255,0.07)', padding: '1px 5px', borderRadius: '4px' }}> attestation </code>
              route. The proxy establishes an aTLS handshake with the agent and verifies the
              hardware attestation report before forwarding. Results are captured in runtime activity logs.
            </p>
          </InfoCard>
        </div>
      </div>
    </div>
  )
}
