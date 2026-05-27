// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import {
  fetchDashboardStats,
  fetchActivityTrends,
  fetchErrorRateTrends,
  fetchModelPerformance,
  fetchTokenBreakdown,
  fetchGuardrailsActivity,
  formatTokens,
  type DashboardStats,
  type ActivityBucket,
  type ModelPerf,
  type TokenBreakdown,
  type GuardrailsStats,
} from '@/lib/dashboard'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

// ── stat card ──────────────────────────────────────────────────────────────

function StatCard({ title, value, sub, accent = false }: { title: string; value: string; sub?: string; accent?: boolean }) {
  return (
    <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '18px 20px' }}>
      <p style={{ margin: '0 0 6px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', fontWeight: '600', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>{title}</p>
      <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '28px', fontWeight: '700', color: accent ? 'var(--accent)' : 'var(--text)', letterSpacing: '-0.03em', lineHeight: 1 }}>{value}</p>
      {sub && <p style={{ margin: '4px 0 0', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{sub}</p>}
    </div>
  )
}

// ── section wrapper ────────────────────────────────────────────────────────

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '20px' }}>
      <p style={{ margin: '0 0 16px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', color: 'var(--text)', letterSpacing: '-0.01em' }}>{title}</p>
      {children}
    </div>
  )
}

// ── activity chart (inline SVG) ────────────────────────────────────────────

function ActivityChart({ buckets }: { buckets: ActivityBucket[] }) {
  if (buckets.length === 0) {
    return <EmptyChart />
  }
  const maxSessions = Math.max(...buckets.map(b => b.sessions), 1)
  const maxTokens = Math.max(...buckets.map(b => b.tokens), 1)
  const w = 600
  const h = 80

  const sessionPts = buckets.map((b, i) => {
    const x = (i / (buckets.length - 1)) * w
    const y = h - (b.sessions / maxSessions) * h
    return `${x},${y}`
  })
  const tokenPts = buckets.map((b, i) => {
    const x = (i / (buckets.length - 1)) * w
    const y = h - (b.tokens / maxTokens) * h
    return `${x},${y}`
  })

  return (
    <div>
      <div style={{ display: 'flex', gap: '16px', marginBottom: '8px' }}>
        <span style={{ display: 'flex', alignItems: 'center', gap: '5px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
          <span style={{ width: '8px', height: '2px', background: '#00d4b4', display: 'inline-block', borderRadius: '1px' }} /> Sessions
        </span>
        <span style={{ display: 'flex', alignItems: 'center', gap: '5px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
          <span style={{ width: '8px', height: '2px', background: '#64a0ff', display: 'inline-block', borderRadius: '1px' }} /> Tokens
        </span>
      </div>
      <svg viewBox={`0 0 ${w} ${h}`} width="100%" height={h} preserveAspectRatio="none" style={{ display: 'block' }}>
        <polyline points={sessionPts.join(' ')} fill="none" stroke="#00d4b4" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
        <polyline points={tokenPts.join(' ')} fill="none" stroke="#64a0ff" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
      </svg>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '4px' }}>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>7d ago</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>now</span>
      </div>
    </div>
  )
}

// ── error rate chart ───────────────────────────────────────────────────────

function ErrorRateChart({ buckets }: { buckets: Array<{ time: string; success: number; clientErrors: number; serverErrors: number }> }) {
  if (buckets.length === 0) return <EmptyChart />
  const max = Math.max(...buckets.map(b => b.success + b.clientErrors + b.serverErrors), 1)
  const w = 600
  const h = 70

  const mkPts = (vals: number[]) =>
    vals.map((v, i) => `${(i / (vals.length - 1)) * w},${h - (v / max) * h}`).join(' ')

  return (
    <div>
      <div style={{ display: 'flex', gap: '16px', marginBottom: '8px' }}>
        {[['#00d4b4', 'Success'], ['#ffb400', 'Client errors'], ['#ff5050', 'Server errors']].map(([color, label]) => (
          <span key={label} style={{ display: 'flex', alignItems: 'center', gap: '5px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
            <span style={{ width: '8px', height: '2px', background: color, display: 'inline-block', borderRadius: '1px' }} /> {label}
          </span>
        ))}
      </div>
      <svg viewBox={`0 0 ${w} ${h}`} width="100%" height={h} preserveAspectRatio="none" style={{ display: 'block' }}>
        <polyline points={mkPts(buckets.map(b => b.success))} fill="none" stroke="#00d4b4" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
        <polyline points={mkPts(buckets.map(b => b.clientErrors))} fill="none" stroke="#ffb400" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
        <polyline points={mkPts(buckets.map(b => b.serverErrors))} fill="none" stroke="#ff5050" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
      </svg>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '4px' }}>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>7d ago</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>now</span>
      </div>
    </div>
  )
}

function EmptyChart() {
  return (
    <div style={{ height: '70px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-dim)' }}>No data yet</span>
    </div>
  )
}

// ── token bar chart ────────────────────────────────────────────────────────

function TokenBar({ item }: { item: TokenBreakdown }) {
  const label = item.model.split(':')[0].split('-').map(s => s[0].toUpperCase() + s.slice(1)).join(' ')
  return (
    <div style={{ marginBottom: '10px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-muted)' }}>{label}</span>
        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>{formatTokens(item.totalTokens)} ({item.percentage.toFixed(0)}%)</span>
      </div>
      <div style={{ height: '6px', background: 'rgba(255,255,255,0.06)', borderRadius: '3px', overflow: 'hidden' }}>
        <div style={{ display: 'flex', height: '100%' }}>
          <div style={{ width: `${item.percentage}%`, background: 'linear-gradient(90deg, #00d4b4, #64a0ff)', borderRadius: '3px', transition: 'width 0.4s ease' }} />
        </div>
      </div>
    </div>
  )
}

// ── guardrails card ────────────────────────────────────────────────────────

function DecisionPill({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <div style={{ textAlign: 'center', padding: '10px', background: 'rgba(255,255,255,0.03)', borderRadius: '8px', border: '1px solid var(--border)' }}>
      <p style={{ margin: '0 0 2px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '16px', fontWeight: '700', color }}>{count.toLocaleString()}</p>
      <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{label}</p>
    </div>
  )
}

function ThreatRow({ label, count }: { label: string; count: number }) {
  if (count === 0) return null
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '5px 0', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-muted)' }}>{label}</span>
      <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '12px', color: '#ff5050' }}>{count.toLocaleString()}</span>
    </div>
  )
}

function GuardrailsCard({ stats }: { stats: GuardrailsStats | null }) {
  const [showThreats, setShowThreats] = useState(false)

  if (!stats) return (
    <Card title="Guardrails Activity">
      <EmptyChart />
    </Card>
  )

  const passRate = stats.totalRequests > 0
    ? ((stats.cleanRequests / stats.totalRequests) * 100).toFixed(1)
    : '100'
  const passColor = Number(passRate) >= 95 ? '#00d4b4' : Number(passRate) >= 80 ? '#ffb400' : '#ff5050'

  const threatCount =
    stats.promptInjection + stats.jailbreakAttempt + stats.toxicContent + stats.offTopicDetected + stats.hallucinationRisk

  return (
    <Card title="Guardrails Activity">
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '8px', marginBottom: '12px' }}>
        <div style={{ textAlign: 'center', padding: '12px', background: 'rgba(255,255,255,0.03)', borderRadius: '8px', border: '1px solid var(--border)', gridColumn: '1 / -1' }}>
          <p style={{ margin: '0 0 2px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '28px', fontWeight: '700', color: passColor }}>{passRate}%</p>
          <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Pass Rate</p>
        </div>
        <DecisionPill label="Content Filtered" count={stats.contentFiltered} color="#ffb400" />
        <DecisionPill label="PII Detected" count={stats.piiDetected} color="#ff5050" />
      </div>

      {stats.guardrailsProcessed > 0 && (
        <div style={{ marginBottom: '12px' }}>
          <p style={{ margin: '0 0 8px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
            Decision Breakdown
            {stats.avgGuardrailsLatencyMs > 0 && (
              <span style={{ marginLeft: '8px', color: 'var(--text-dim)' }}>avg {stats.avgGuardrailsLatencyMs.toFixed(1)}ms</span>
            )}
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '6px' }}>
            <DecisionPill label="Allow" count={stats.decisionsAllow} color="#00d4b4" />
            <DecisionPill label="Block" count={stats.decisionsBlock} color="#ff5050" />
            <DecisionPill label="Modify" count={stats.decisionsModify} color="#ffb400" />
          </div>
        </div>
      )}

      {threatCount > 0 && (
        <div>
          <button
            onClick={() => setShowThreats(s => !s)}
            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '4px 0', width: '100%', textAlign: 'left', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
          >
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff5050', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Threat Detection ({threatCount})</span>
            <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)' }}>{showThreats ? 'hide' : 'show'}</span>
          </button>
          {showThreats && (
            <div style={{ marginTop: '6px' }}>
              <ThreatRow label="Prompt Injection" count={stats.promptInjection} />
              <ThreatRow label="Jailbreak Attempt" count={stats.jailbreakAttempt} />
              <ThreatRow label="Toxic Content" count={stats.toxicContent} />
              <ThreatRow label="Off-Topic" count={stats.offTopicDetected} />
              <ThreatRow label="Hallucination Risk" count={stats.hallucinationRisk} />
            </div>
          )}
        </div>
      )}

      {stats.totalRequests === 0 && (
        <p style={{ margin: '8px 0 0', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-dim)' }}>
          No activity in the last 7 days.
        </p>
      )}
    </Card>
  )
}

// ── model performance table ────────────────────────────────────────────────

function ModelPerfTable({ models }: { models: ModelPerf[] }) {
  if (models.length === 0) return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '80px' }}>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-dim)' }}>No model activity in last 7 days</span>
    </div>
  )
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid var(--border)' }}>
            {['Model', 'Requests', 'Avg Latency', 'Avg Input', 'Avg Output'].map(h => (
              <th key={h} style={{ padding: '6px 8px', textAlign: 'left', color: 'var(--text-dim)', fontWeight: '600', fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.05em', whiteSpace: 'nowrap' }}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {models.map(m => {
            const label = m.model.split(':')[0].split('-').map(s => s[0].toUpperCase() + s.slice(1)).join(' ')
            return (
              <tr key={m.model} style={{ borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                <td style={{ padding: '8px', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>{label}</td>
                <td style={{ padding: '8px', color: 'var(--text-dim)' }}>{m.requestCount.toLocaleString()}</td>
                <td style={{ padding: '8px', color: m.avgLatencyMs > 0 ? 'var(--text-muted)' : 'var(--text-dim)' }}>
                  {m.avgLatencyMs > 0 ? `${m.avgLatencyMs}ms` : '—'}
                </td>
                <td style={{ padding: '8px', color: 'var(--text-dim)' }}>{formatTokens(m.avgInputTokens)}</td>
                <td style={{ padding: '8px', color: 'var(--text-dim)' }}>{formatTokens(m.avgOutputTokens)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

// ── spinner ────────────────────────────────────────────────────────────────

function Spinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '40px' }}>
      <div style={{ width: '20px', height: '20px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
    </div>
  )
}

// ── main page ──────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [activity, setActivity] = useState<ActivityBucket[]>([])
  const [errorRate, setErrorRate] = useState<Array<{ time: string; success: number; clientErrors: number; serverErrors: number }>>([])
  const [modelPerf, setModelPerf] = useState<ModelPerf[]>([])
  const [tokenBreakdown, setTokenBreakdown] = useState<TokenBreakdown[]>([])
  const [guardrails, setGuardrails] = useState<GuardrailsStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!tokens?.accessToken || !activeDomain?.id) return
    setLoading(true)
    setError(null)
    try {
      const [s, act, err, mp, tb, gr] = await Promise.all([
        fetchDashboardStats(activeDomain.id, tokens.accessToken),
        fetchActivityTrends(activeDomain.id, tokens.accessToken),
        fetchErrorRateTrends(activeDomain.id, tokens.accessToken),
        fetchModelPerformance(activeDomain.id, tokens.accessToken),
        fetchTokenBreakdown(activeDomain.id, tokens.accessToken),
        fetchGuardrailsActivity(activeDomain.id, tokens.accessToken),
      ])
      setStats(s)
      setActivity(act)
      setErrorRate(err)
      setModelPerf(mp)
      setTokenBreakdown(tb)
      setGuardrails(gr)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load dashboard')
    } finally {
      setLoading(false)
    }
  }, [tokens?.accessToken, activeDomain?.id])

  useEffect(() => { void load() }, [load])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>
            Dashboard
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            {activeDomain ? `${activeDomain.name} · last 7 days` : 'Select a domain to see analytics'}
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

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {!activeDomain && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,180,0,0.06)', border: '1px solid rgba(255,180,0,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ffb400' }}>
            Select a domain from the Domains page to view analytics.
          </div>
        )}

        {activeDomain && loading && <Spinner />}

        {activeDomain && !loading && error && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {error}
          </div>
        )}

        {activeDomain && !loading && !error && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Stats row */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '14px' }}>
              <StatCard title="Conversations Today" value={(stats?.conversationsToday ?? 0).toString()} sub="Unique sessions" accent={!!stats?.conversationsToday} />
              <StatCard title="Active Models" value={(stats?.activeModels ?? 0).toString()} sub="Models used today" accent={!!stats?.activeModels} />
              <StatCard title="Tokens Today" value={formatTokens(stats?.tokensToday ?? 0)} sub="Input + output" accent={!!stats?.tokensToday} />
            </div>

            {/* Charts row */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
              <Card title="Activity Trends (7d)">
                <ActivityChart buckets={activity} />
              </Card>
              <Card title="Error Rate Trends (7d)">
                <ErrorRateChart buckets={errorRate} />
              </Card>
            </div>

            {/* Model Performance + Guardrails row */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: '14px' }}>
              <Card title="Model Performance (7d)">
                <ModelPerfTable models={modelPerf} />
              </Card>
              <GuardrailsCard stats={guardrails} />
            </div>

            {/* Token usage */}
            {tokenBreakdown.length > 0 && (
              <Card title="Token Usage by Model (7d)">
                <div style={{ maxWidth: '600px' }}>
                  {tokenBreakdown.map(item => (
                    <TokenBar key={item.model} item={item} />
                  ))}
                </div>
              </Card>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
