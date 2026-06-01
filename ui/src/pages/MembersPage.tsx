// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { listDomainMembers, listUsers, listDomainRoles, sendInvitation } from '@/lib/platform/service'
import type { MemberRoles, User, DomainRole } from '@/lib/platform/service'
import type { AppContext } from '@/types'
import UserMenu from '@/components/UserMenu'

function NoDomainBanner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px' }}>
      <div style={{ textAlign: 'center', gap: '12px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none">
          <circle cx="12" cy="12" r="10" stroke="var(--text-dim)" strokeWidth="1.5" />
          <path d="M12 8v4M12 16h.01" stroke="var(--text-dim)" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
        <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>
          Select a domain first to view its members.
        </p>
      </div>
    </div>
  )
}

function InviteModal({
  domainID,
  roles,
  onClose,
  onSent,
}: {
  domainID: string
  roles: DomainRole[]
  onClose: () => void
  onSent: () => void
}) {
  const { tokens } = useAuth()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [searching, setSearching] = useState(false)
  const [selected, setSelected] = useState<User | null>(null)
  const [roleID, setRoleID] = useState(roles[0]?.id ?? '')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [allUsers, setAllUsers] = useState<User[]>([])

  useEffect(() => {
    if (!tokens?.accessToken) return
    listUsers(tokens.accessToken)
      .then(setAllUsers)
      .catch(() => {})
  }, [tokens?.accessToken])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (!query.trim()) { setResults([]); return }
    setSearching(true)
    const q = query.toLowerCase()
    const filtered = allUsers.filter(u =>
      u.email?.toLowerCase().includes(q) ||
      (u.first_name + ' ' + u.last_name).toLowerCase().includes(q) ||
      (u.credentials as { username?: string } | undefined)?.username?.toLowerCase().includes(q)
    )
    setResults(filtered.slice(0, 8))
    setSearching(false)
  }, [query, allUsers])

  async function handleSend() {
    if (!tokens?.accessToken || !selected?.id || !roleID) return
    setSending(true)
    setError(null)
    try {
      await sendInvitation(selected.id, domainID, roleID, tokens.accessToken)
      onSent()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send invitation')
    } finally {
      setSending(false)
    }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }} onClick={onClose}>
      <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '14px', padding: '28px', width: '400px', boxShadow: '0 24px 80px rgba(0,0,0,0.5)' }} onClick={e => e.stopPropagation()}>
        <h2 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '18px', color: 'var(--text)', margin: '0 0 4px', letterSpacing: '-0.02em' }}>
          Invite member
        </h2>
        <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: '0 0 20px' }}>
          Send an invitation to a user to join this domain.
        </p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
          {/* User search */}
          <div>
            <label style={{ display: 'block', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', marginBottom: '5px' }}>
              Search user
            </label>
            {selected ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '8px 12px', background: 'rgba(0,212,180,0.06)', border: '1px solid rgba(0,212,180,0.25)', borderRadius: '8px' }}>
                <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: 'rgba(0,212,180,0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '11px', color: 'var(--accent)' }}>
                    {selected.email?.[0]?.toUpperCase() ?? '?'}
                  </span>
                </div>
                <div style={{ flex: 1 }}>
                  <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '600', color: 'var(--text)' }}>
                    {selected.first_name ? `${selected.first_name} ${selected.last_name ?? ''}`.trim() : selected.email}
                  </p>
                  <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{selected.email}</p>
                </div>
                <button onClick={() => setSelected(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-dim)', padding: '2px' }}>
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M2 2l10 10M12 2L2 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
                  </svg>
                </button>
              </div>
            ) : (
              <div style={{ position: 'relative' }}>
                <input
                  type="text"
                  placeholder="Search by name or email…"
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  autoFocus
                  style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
                />
                {(results.length > 0 || searching) && (
                  <div style={{ position: 'absolute', top: 'calc(100% + 4px)', left: 0, right: 0, background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '8px', overflow: 'hidden', boxShadow: '0 8px 24px rgba(0,0,0,0.4)', zIndex: 10 }}>
                    {results.map(u => (
                      <button
                        key={u.id}
                        onClick={() => { setSelected(u); setQuery(''); setResults([]) }}
                        style={{ display: 'flex', alignItems: 'center', gap: '10px', width: '100%', padding: '9px 12px', background: 'none', border: 'none', cursor: 'pointer', textAlign: 'left' }}
                        onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,255,255,0.05)' }}
                        onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}
                      >
                        <div style={{ width: '26px', height: '26px', borderRadius: '50%', background: 'rgba(0,212,180,0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                          <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '10px', color: 'var(--accent)' }}>
                            {u.email?.[0]?.toUpperCase() ?? '?'}
                          </span>
                        </div>
                        <div>
                          <p style={{ margin: 0, fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '600', color: 'var(--text)' }}>
                            {u.first_name ? `${u.first_name} ${u.last_name ?? ''}`.trim() : u.email}
                          </p>
                          <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{u.email}</p>
                        </div>
                      </button>
                    ))}
                    {results.length === 0 && !searching && (
                      <p style={{ padding: '10px 12px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-dim)', margin: 0 }}>No users found</p>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Role selection */}
          {roles.length > 0 && (
            <div>
              <label style={{ display: 'block', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '12px', color: 'var(--text)', marginBottom: '5px' }}>
                Role
              </label>
              <select
                value={roleID}
                onChange={e => setRoleID(e.target.value)}
                style={{ width: '100%', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '9px 12px', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', outline: 'none', boxSizing: 'border-box' }}
              >
                {roles.map(r => (
                  <option key={r.id} value={r.id} style={{ background: 'var(--card-bg)' }}>{r.name}</option>
                ))}
              </select>
            </div>
          )}

          {error && (
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: '#ff6b6b', margin: 0, padding: '7px 10px', background: 'rgba(255,107,107,0.08)', borderRadius: '6px', border: '1px solid rgba(255,107,107,0.2)' }}>
              {error}
            </p>
          )}

          <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
            <button onClick={onClose} style={{ padding: '8px 16px', borderRadius: '8px', border: '1px solid var(--border)', background: 'transparent', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '600', cursor: 'pointer' }}>
              Cancel
            </button>
            <button
              onClick={handleSend}
              disabled={!selected || sending}
              style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: !selected || sending ? 'rgba(0,212,180,0.4)' : 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: !selected || sending ? 'not-allowed' : 'pointer' }}
            >
              {sending ? 'Sending…' : 'Send invitation'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function MemberRow({ member }: { member: MemberRoles }) {
  const roleName = member.roles?.[0]?.role_name ?? '—'
  const id = member.member_id ?? '?'
  const short = id.slice(0, 8)

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '11px 16px', borderBottom: '1px solid var(--border)' }}>
      <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'rgba(0,212,180,0.12)', border: '1px solid rgba(0,212,180,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '12px', color: 'var(--accent)' }}>
          {short[0]?.toUpperCase() ?? '?'}
        </span>
      </div>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <p style={{ margin: 0, fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {id}
        </p>
      </div>
      <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '600', color: 'var(--text-muted)', padding: '3px 8px', background: 'rgba(255,255,255,0.04)', borderRadius: '6px', border: '1px solid var(--border)', flexShrink: 0 }}>
        {roleName}
      </span>
    </div>
  )
}

export default function MembersPage() {
  const { tokens } = useAuth()
  const { activeDomain } = useOutletContext<AppContext>()

  const [members, setMembers] = useState<MemberRoles[]>([])
  const [roles, setRoles] = useState<DomainRole[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showInvite, setShowInvite] = useState(false)

  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const load = useCallback(async () => {
    if (!tokens?.accessToken || !activeDomain?.id) return
    setLoading(true)
    setError(null)
    try {
      const [res, domainRoles] = await Promise.all([
        listDomainMembers(activeDomain.id, tokens.accessToken),
        listDomainRoles(activeDomain.id, tokens.accessToken),
      ])
      setMembers(res.members)
      setRoles(domainRoles)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load members')
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
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '20px', color: 'var(--text)', margin: '0 0 2px', letterSpacing: '-0.02em' }}>Members</h1>
          {activeDomain && (
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: 0 }}>
              {activeDomain.name} · {members.length} member{members.length !== 1 ? 's' : ''}
            </p>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          {activeDomain && (
            <button
              onClick={() => setShowInvite(true)}
              style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 14px', background: 'var(--accent)', border: 'none', borderRadius: '8px', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: 'pointer' }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M7 1v12M1 7h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
              </svg>
              Invite member
            </button>
          )}
          <UserMenu />
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 28px' }}>
        {!activeDomain && <NoDomainBanner />}

        {activeDomain && loading && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '200px' }}>
            <div style={{ width: '24px', height: '24px', border: '2px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          </div>
        )}

        {activeDomain && !loading && error && (
          <div style={{ padding: '14px 16px', background: 'rgba(255,107,107,0.08)', border: '1px solid rgba(255,107,107,0.2)', borderRadius: '10px', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b' }}>
            {error}
          </div>
        )}

        {activeDomain && !loading && !error && members.length === 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '300px', gap: '12px' }}>
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>No members yet</p>
            <button onClick={() => setShowInvite(true)} style={{ padding: '7px 16px', background: 'var(--accent)', border: 'none', borderRadius: '8px', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', cursor: 'pointer' }}>
              Invite the first member
            </button>
          </div>
        )}

        {activeDomain && !loading && !error && members.length > 0 && (
          <div style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', overflow: 'hidden' }}>
            <div style={{ display: 'flex', alignItems: 'center', padding: '10px 16px', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)' }}>
              <span style={{ flex: 1, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', fontWeight: '600', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Member ID</span>
              <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', fontWeight: '600', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Role</span>
            </div>
            {members.map((m, i) => <MemberRow key={m.member_id ?? i} member={m} />)}
          </div>
        )}
      </div>

      {showInvite && activeDomain && (
        <InviteModal
          domainID={activeDomain.id}
          roles={roles}
          onClose={() => setShowInvite(false)}
          onSent={() => { setShowInvite(false); void load() }}
        />
      )}
    </div>
  )
}
