// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate, useOutletContext } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { listDomains, createDomain, deleteDomain, toRoute } from '@/lib/platform/service'
import type { Domain } from '@/lib/platform/service'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

const LAST_DOMAIN_KEY = 'cube_active_domain'
const DOMAINS_STALE_TIME_MS = 30_000

function domainsQueryKey(userID: string) {
  return ['domains', userID] as const
}

const statusColors: Record<string, { bg: string; color: string }> = {
  enabled:  { bg: 'rgba(0,212,180,0.1)',  color: '#00d4b4' },
  disabled: { bg: 'rgba(255,180,0,0.1)',  color: '#ffb400' },
  frozen:   { bg: 'rgba(255,80,80,0.1)',   color: '#ff5050' },
}

function StatusBadge({ status }: { status?: string }) {
  const s = status ?? 'enabled'
  const c = statusColors[s] ?? statusColors['enabled']
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '5px', background: c.bg, color: c.color, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '3px 8px', borderRadius: '20px', fontWeight: '500' }}>
      <span style={{ width: '5px', height: '5px', borderRadius: '50%', background: c.color }} />
      {s}
    </span>
  )
}

function DomainIcon({ name }: { name: string }) {
  return (
    <div style={{ width: '36px', height: '36px', borderRadius: '9px', background: 'rgba(0,212,180,0.12)', border: '1px solid rgba(0,212,180,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '14px', color: 'var(--accent)' }}>
        {name[0]?.toUpperCase() ?? '?'}
      </span>
    </div>
  )
}

function loadLastDomainID(): string {
  try {
    const raw = localStorage.getItem(LAST_DOMAIN_KEY)
    if (!raw) return ''
    const parsed = JSON.parse(raw) as { id?: unknown }
    return typeof parsed.id === 'string' ? parsed.id : ''
  } catch {
    return ''
  }
}

function CreateDomainModal({ onClose, onCreated }: { onClose: () => void; onCreated: (d: Domain) => Promise<void> }) {
  const { tokens } = useAuth()
  const [name, setName] = useState('')
  const [route, setRoute] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fieldErrors, setFieldErrors] = useState<{ name?: string; route?: string }>({})
  const submitting = useRef(false)

  function handleNameChange(value: string) {
    setName(value)
    if (fieldErrors.name) setFieldErrors(prev => ({ ...prev, name: undefined }))
  }

  function handleRouteChange(value: string) {
    setRoute(toRoute(value))
    if (fieldErrors.route) setFieldErrors(prev => ({ ...prev, route: undefined }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const nextErrors: { name?: string; route?: string } = {}
    if (!name.trim()) nextErrors.name = 'Domain name is required.'
    if (!route.trim()) nextErrors.route = 'Route is required.'
    setFieldErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0 || submitting.current || !tokens?.accessToken) return

    submitting.current = true
    setLoading(true)
    setError(null)
    try {
      const domain = await createDomain(name.trim(), route.trim(), tokens.accessToken)
      await onCreated(domain)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create domain')
    } finally {
      submitting.current = false
      setLoading(false)
    }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }} onClick={onClose}>
      <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '14px', padding: '28px', width: '380px', boxShadow: '0 24px 80px rgba(0,0,0,0.5)' }} onClick={e => e.stopPropagation()}>
        <h2 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '18px', color: 'var(--text)', margin: '0 0 4px', letterSpacing: '-0.02em' }}>
          Create domain
        </h2>
        <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: '0 0 20px' }}>
          A domain is a workspace that groups your resources.
        </p>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          <div>
            <label style={{ display: 'block', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', marginBottom: '5px' }}>
              Domain name *
            </label>
            <input
              type="text"
              placeholder="e.g. My Workspace"
              value={name}
              onChange={e => handleNameChange(e.target.value)}
              autoFocus
              style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: `1px solid ${fieldErrors.name ? 'rgba(255,107,107,0.5)' : 'var(--border)'}`, borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', outline: 'none', boxSizing: 'border-box' }}
            />
            {fieldErrors.name && (
              <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff6b6b', margin: '5px 0 0' }}>{fieldErrors.name}</p>
            )}
          </div>

          <div>
            <label style={{ display: 'block', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', marginBottom: '5px' }}>
              Route * <span style={{ fontWeight: '400', color: 'var(--text-muted)' }}>(must be unique)</span>
            </label>
            <input
              type="text"
              placeholder="e.g. my-workspace"
              value={route}
              onChange={e => handleRouteChange(e.target.value)}
              style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: `1px solid ${fieldErrors.route ? 'rgba(255,107,107,0.5)' : 'var(--border)'}`, borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
            />
            {fieldErrors.route ? (
              <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff6b6b', margin: '5px 0 0' }}>{fieldErrors.route}</p>
            ) : (
              <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', margin: '5px 0 0' }}>
                Use lowercase letters, numbers, and hyphens only.
              </p>
            )}
          </div>

          {error && (
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: '#ff6b6b', margin: 0, padding: '7px 10px', background: 'rgba(255,107,107,0.08)', borderRadius: '6px', border: '1px solid rgba(255,107,107,0.2)' }}>
              {error}
            </p>
          )}

          <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
            <button
              type="button"
              onClick={onClose}
              style={{ padding: '8px 16px', borderRadius: '8px', border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '600', cursor: 'pointer' }}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: loading ? 'rgba(0,212,180,0.4)' : 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: loading ? 'not-allowed' : 'pointer' }}
            >
              {loading ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function DomainsPage() {
  const { tokens, user } = useAuth()
  const accessToken = tokens?.accessToken
  const userID = user?.id ?? ''
  const location = useLocation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { activeDomain, setActiveDomain } = useOutletContext<AppContext>()

  const [actionError, setActionError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [lastDomainID] = useState(loadLastDomainID)
  const redirectPath = (() => {
    const state = location.state as { from?: string } | null
    const from = state?.from
    return from && from !== '/domains' ? from : '/dashboard'
  })()

  const {
    data: domains = [],
    error: loadError,
    isPending: loading,
  } = useQuery({
    queryKey: domainsQueryKey(userID),
    queryFn: () => listDomains(accessToken!),
    enabled: Boolean(accessToken && userID),
    staleTime: DOMAINS_STALE_TIME_MS,
  })

  useEffect(() => {
    if (!loading && !loadError && activeDomain && !domains.some(domain => domain.id === activeDomain.id)) {
      setActiveDomain(null)
    }
  }, [activeDomain, domains, loadError, loading, setActiveDomain])

  function handleSelect(domain: Domain) {
    const id = domain.id ?? ''
    if (!id) return
    setActiveDomain({
      id,
      name: domain.name ?? '',
      route: domain.route,
      status: domain.status as string | undefined,
    })
    navigate(redirectPath, { replace: true })
  }

  async function handleDelete(domain: Domain) {
    if (!tokens?.accessToken) return
    if (!window.confirm(`Delete domain "${domain.name}"? This cannot be undone.`)) return
    setActionError(null)
    try {
      await deleteDomain(domain.id ?? '', tokens.accessToken)
      queryClient.setQueryData<Domain[]>(
        domainsQueryKey(userID),
        current => current?.filter(d => d.id !== domain.id) ?? [],
      )
      if (activeDomain?.id === domain.id) setActiveDomain(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete domain')
    }
  }

  async function handleCreated(domain: Domain) {
    queryClient.setQueryData<Domain[]>(
      domainsQueryKey(userID),
      current => current?.some(item => item.id === domain.id) ? current : [...(current ?? []), domain],
    )
    setShowCreate(false)
    handleSelect(domain)
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '20px 28px 16px', borderBottom: '1px solid var(--border)', flexShrink: 0 }}>
        <div>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>
            Domains
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
            Workspaces that organize your resources
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button
            onClick={() => setShowCreate(true)}
            style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 14px', background: 'var(--accent)', border: 'none', borderRadius: '8px', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: 'pointer' }}
          >
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M7 1v12M1 7h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            </svg>
            New domain
          </button>
          <UserMenu />
        </div>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {loading && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '200px' }}>
            <div style={{ width: '24px', height: '24px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          </div>
        )}

        {!loading && loadError && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {loadError instanceof Error ? loadError.message : 'Failed to load domains'}
          </div>
        )}

        {actionError && (
          <div style={{ padding: '14px 16px', marginBottom: '16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {actionError}
          </div>
        )}

        {!loading && !loadError && domains.length === 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '300px', gap: '12px' }}>
            <div style={{ width: '48px', height: '48px', borderRadius: '12px', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <svg width="22" height="22" viewBox="0 0 20 20" fill="none">
                <circle cx="10" cy="10" r="7.5" stroke="var(--text-dim)" strokeWidth="1.5" />
                <path d="M10 2.5C10 2.5 7 6 7 10s3 7.5 3 7.5M10 2.5c0 0 3 3.5 3 7.5s-3 7.5-3 7.5" stroke="var(--text-dim)" strokeWidth="1.5" />
                <path d="M2.5 10h15" stroke="var(--text-dim)" strokeWidth="1.5" strokeLinecap="round" />
              </svg>
            </div>
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>No domains yet</p>
            <button
              onClick={() => setShowCreate(true)}
              style={{ padding: '7px 16px', background: 'var(--accent)', border: 'none', borderRadius: '8px', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: 'pointer' }}
            >
              Create your first domain
            </button>
          </div>
        )}

        {!loading && !loadError && domains.length > 0 && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '12px' }}>
            {domains.map(domain => {
              const isActive = activeDomain?.id === domain.id
              const isLast = !activeDomain && lastDomainID === domain.id
              return (
                <div
                  key={domain.id}
                  style={{
                    background: isActive || isLast ? 'rgba(0,212,180,0.05)' : 'var(--card-bg)',
                    border: `1px solid ${isActive || isLast ? 'rgba(0,212,180,0.3)' : 'var(--border)'}`,
                    borderRadius: '12px',
                    padding: '16px',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '12px',
                    cursor: 'pointer',
                    transition: 'border-color 0.15s ease, background 0.15s ease',
                  }}
                  onClick={() => handleSelect(domain)}
                  onMouseEnter={e => { if (!isActive && !isLast) (e.currentTarget as HTMLDivElement).style.borderColor = 'rgba(255,255,255,0.12)' }}
                  onMouseLeave={e => { if (!isActive && !isLast) (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)' }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <DomainIcon name={domain.name ?? '?'} />
                    <div style={{ flex: 1, overflow: 'hidden' }}>
                      <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '14px', color: 'var(--text)', margin: '0 0 2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {domain.name}
                      </p>
                      {domain.route && (
                        <p style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', margin: 0 }}>
                          {domain.route}
                        </p>
                      )}
                    </div>
                    {(isActive || isLast) && (
                      <div style={{ flexShrink: 0, width: '18px', height: '18px', borderRadius: '50%', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                          <path d="M2 5l2.5 2.5L8 3" stroke="#070c16" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                        </svg>
                      </div>
                    )}
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <StatusBadge status={domain.status as string | undefined} />
                    <div style={{ display: 'flex', gap: '6px' }}>
                      <button
                        onClick={e => { e.stopPropagation(); handleSelect(domain) }}
                        style={{ padding: '4px 10px', border: `1px solid ${isActive || isLast ? 'rgba(0,212,180,0.4)' : 'var(--border)'}`, borderRadius: '6px', background: isActive || isLast ? 'rgba(0,212,180,0.1)' : 'transparent', color: isActive || isLast ? 'var(--accent)' : 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', fontWeight: '600', cursor: 'pointer' }}
                      >
                        {isActive ? 'Active' : isLast ? 'Last used' : 'Select'}
                      </button>
                      <button
                        onClick={e => { e.stopPropagation(); void handleDelete(domain) }}
                        style={{ padding: '4px 10px', border: '1px solid rgba(255,80,80,0.3)', borderRadius: '6px', background: 'transparent', color: '#ff5050', fontFamily: 'Space Grotesk, sans-serif', fontSize: '11px', fontWeight: '600', cursor: 'pointer' }}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {showCreate && (
        <CreateDomainModal
          onClose={() => setShowCreate(false)}
          onCreated={handleCreated}
        />
      )}
    </div>
  )
}
