// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import {
  fetchGuardrailsActivity,
  type GuardrailsStats,
} from '@/lib/dashboard'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

function Spinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '60px' }}>
      <div style={{ width: '20px', height: '20px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
    </div>
  )
}

function BigStat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '20px', textAlign: 'center' }}>
      <p style={{ margin: '0 0 6px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '32px', fontWeight: '700', color: color ?? 'var(--text)', letterSpacing: '-0.03em', lineHeight: 1 }}>{value}</p>
      <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>{label}</p>
    </div>
  )
}

function DecisionBar({ label, count, total, color }: { label: string; count: number; total: number; color: string }) {
  const pct = total > 0 ? (count / total) * 100 : 0
  return (
    <div style={{ marginBottom: '14px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '5px' }}>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', fontWeight: '500' }}>{label}</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color }}>
          {count.toLocaleString()} <span style={{ color: 'var(--text-dim)' }}>({pct.toFixed(1)}%)</span>
        </span>
      </div>
      <div style={{ height: '7px', background: 'rgba(255,255,255,0.06)', borderRadius: '4px', overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: '4px', transition: 'width 0.5s ease' }} />
      </div>
    </div>
  )
}

function ThreatBar({ label, count, max }: { label: string; count: number; max: number }) {
  if (count === 0) return null
  const pct = max > 0 ? (count / max) * 100 : 0
  return (
    <div style={{ marginBottom: '12px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)' }}>{label}</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color: '#ff5050' }}>{count.toLocaleString()}</span>
      </div>
      <div style={{ height: '5px', background: 'rgba(255,255,255,0.06)', borderRadius: '3px', overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: 'linear-gradient(90deg, #ff5050, #ff8080)', borderRadius: '3px', transition: 'width 0.5s ease' }} />
      </div>
    </div>
  )
}

function Card({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '20px' }}>
      <div style={{ marginBottom: '16px' }}>
        <p style={{ margin: '0 0 2px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', fontWeight: '700', color: 'var(--text)', letterSpacing: '-0.01em' }}>{title}</p>
        {subtitle && <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{subtitle}</p>}
      </div>
      {children}
    </div>
  )
}

function EmptyState({ message }: { message: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '32px' }}>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-dim)' }}>{message}</span>
    </div>
  )
}

function GuardrailsContent({ stats }: { stats: GuardrailsStats }) {
  const passRate = stats.totalRequests > 0
    ? (stats.cleanRequests / stats.totalRequests) * 100
    : 100
  const passColor = passRate >= 95 ? '#00d4b4' : passRate >= 80 ? '#ffb400' : '#ff5050'

  const decisionTotal = stats.decisionsAllow + stats.decisionsBlock + stats.decisionsModify
  const threatTotal =
    stats.promptInjection + stats.jailbreakAttempt + stats.toxicContent +
    stats.offTopicDetected + stats.hallucinationRisk
  const maxThreat = Math.max(
    stats.promptInjection, stats.jailbreakAttempt, stats.toxicContent,
    stats.offTopicDetected, stats.hallucinationRisk,
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
      {/* Top stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px' }}>
        <BigStat label="Pass Rate" value={`${passRate.toFixed(1)}%`} color={passColor} />
        <BigStat label="Total Requests" value={stats.totalRequests.toLocaleString()} />
        <BigStat label="Content Filtered" value={stats.contentFiltered.toLocaleString()} color={stats.contentFiltered > 0 ? '#ffb400' : undefined} />
        <BigStat label="PII Detected" value={stats.piiDetected.toLocaleString()} color={stats.piiDetected > 0 ? '#ff5050' : undefined} />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
        {/* Decision breakdown */}
        <Card title="Decision Breakdown" subtitle={stats.avgGuardrailsLatencyMs > 0 ? `avg latency ${stats.avgGuardrailsLatencyMs.toFixed(1)}ms` : undefined}>
          {decisionTotal === 0 ? (
            <EmptyState message="No guardrails decisions recorded in the last 7 days" />
          ) : (
            <>
              <DecisionBar label="Allow" count={stats.decisionsAllow} total={decisionTotal} color="#00d4b4" />
              <DecisionBar label="Block" count={stats.decisionsBlock} total={decisionTotal} color="#ff5050" />
              <DecisionBar label="Modify" count={stats.decisionsModify} total={decisionTotal} color="#ffb400" />
            </>
          )}
        </Card>

        {/* Threat detection */}
        <Card title="Threat Detection" subtitle={`${threatTotal.toLocaleString()} threats detected in last 7 days`}>
          {threatTotal === 0 ? (
            <EmptyState message="No threats detected in the last 7 days" />
          ) : (
            <>
              <ThreatBar label="Prompt Injection" count={stats.promptInjection} max={maxThreat} />
              <ThreatBar label="Jailbreak Attempt" count={stats.jailbreakAttempt} max={maxThreat} />
              <ThreatBar label="Toxic Content" count={stats.toxicContent} max={maxThreat} />
              <ThreatBar label="Off-Topic" count={stats.offTopicDetected} max={maxThreat} />
              <ThreatBar label="Hallucination Risk" count={stats.hallucinationRisk} max={maxThreat} />
            </>
          )}
        </Card>
      </div>

      {/* Pass rate gauge */}
      <Card title="Coverage" subtitle="Requests processed by guardrails">
        {stats.guardrailsProcessed === 0 ? (
          <EmptyState message="No guardrails coverage data available" />
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '6px' }}>
                <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)' }}>Guardrails processed</span>
                <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color: 'var(--accent)' }}>
                  {stats.guardrailsProcessed.toLocaleString()} / {stats.totalRequests.toLocaleString()}
                </span>
              </div>
              <div style={{ height: '8px', background: 'rgba(255,255,255,0.06)', borderRadius: '4px', overflow: 'hidden' }}>
                <div style={{ width: `${stats.totalRequests > 0 ? (stats.guardrailsProcessed / stats.totalRequests) * 100 : 0}%`, height: '100%', background: 'linear-gradient(90deg, var(--accent), #64a0ff)', borderRadius: '4px', transition: 'width 0.5s ease' }} />
              </div>
            </div>
            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '22px', fontWeight: '700', color: 'var(--accent)', flexShrink: 0 }}>
              {stats.totalRequests > 0 ? ((stats.guardrailsProcessed / stats.totalRequests) * 100).toFixed(0) : 0}%
            </div>
          </div>
        )}
      </Card>

      {stats.totalRequests === 0 && (
        <div style={{ padding: '14px 16px', background: 'rgba(255,180,0,0.06)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ffb400' }}>
          No guardrails activity recorded in the last 7 days. Make sure the guardrails service is running and connected.
        </div>
      )}
    </div>
  )
}

export default function GuardrailsPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [stats, setStats] = useState<GuardrailsStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const load = useCallback(async () => {
    if (!tokens?.accessToken || !activeDomain?.id) return
    setLoading(true)
    setError(null)
    try {
      const gr = await fetchGuardrailsActivity(activeDomain.id, tokens.accessToken)
      setStats(gr)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load guardrails data')
    } finally {
      setLoading(false)
    }
  }, [tokens?.accessToken, activeDomain?.id])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { void load() }, [load])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>
            Guardrails
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            {activeDomain ? `${activeDomain.name} · last 7 days` : 'Select a domain to view guardrails activity'}
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          {activeDomain && (
            <button
              onClick={() => void load()}
              style={{ background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border)', borderRadius: '7px', padding: '6px 12px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', cursor: 'pointer' }}
            >
              Refresh
            </button>
          )}
          <UserMenu />
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {!activeDomain && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,180,0,0.06)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ffb400' }}>
            Select a domain from the Domains page to view guardrails activity.
          </div>
        )}

        {activeDomain && loading && <Spinner />}

        {activeDomain && !loading && error && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {error}
          </div>
        )}

        {activeDomain && !loading && !error && stats && (
          <GuardrailsContent stats={stats} />
        )}
      </div>
    </div>
  )
}
