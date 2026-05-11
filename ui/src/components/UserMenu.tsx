import { useState, useEffect, useRef } from 'react'
import { useAuth } from '@/hooks/useAuth'

function initials(user: { firstName?: string; lastName?: string; email: string }): string {
  const f = user.firstName?.[0] ?? ''
  const l = user.lastName?.[0] ?? ''
  if (f && l) return `${f}${l}`.toUpperCase()
  return user.email.slice(0, 2).toUpperCase()
}

function displayName(user: { firstName?: string; lastName?: string; email: string }): string {
  const parts = [user.firstName, user.lastName].filter(Boolean)
  return parts.length > 0 ? parts.join(' ') : user.email
}

export default function UserMenu() {
  const [open, setOpen] = useState(false)
  const { user, logout } = useAuth()
  const containerRef = useRef<HTMLDivElement>(null)

  // Close on outside click — bubble phase, fires AFTER the button's onClick
  useEffect(() => {
    if (!open) return
    const handleOutsideClick = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    window.addEventListener('click', handleOutsideClick)
    return () => window.removeEventListener('click', handleOutsideClick)
  }, [open])

  if (!user) return null

  const abbr = initials(user)
  const name = displayName(user)

  return (
    <div ref={containerRef} style={{ position: 'relative', flexShrink: 0 }}>
      <button
        onClick={() => setOpen(o => !o)}
        style={{ padding: '9px 13px', borderRadius: '8px', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '13px', color: '#070c16', border: 'none', cursor: 'pointer' }}
      >
        {abbr}
      </button>
      {open && (
        <div style={{ position: 'absolute', top: 'calc(100% + 8px)', right: 0, background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '12px', padding: '16px 18px', width: '230px', boxShadow: '0 16px 48px rgba(0,0,0,0.6)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '14px', paddingBottom: '14px', borderBottom: '1px solid var(--border)' }}>
            <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '14px', color: '#070c16', flexShrink: 0 }}>{abbr}</div>
            <div>
              <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '14px', color: 'var(--text)' }}>{name}</div>
              {user.role && (
                <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--accent)', marginTop: '2px', textTransform: 'capitalize' }}>{user.role}</div>
              )}
            </div>
          </div>
          {[
            { label: 'EMAIL', value: user.email },
            { label: 'USERNAME', value: user.username },
            ...(user.role ? [{ label: 'ROLE', value: user.role }] : []),
          ].map(({ label, value }) => (
            <div key={label} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: '10px' }}>
              <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: 'var(--text-dim)', letterSpacing: '0.08em' }}>{label}</span>
              <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text-muted)', fontWeight: '500', maxWidth: '130px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{value}</span>
            </div>
          ))}
          <div style={{ borderTop: '1px solid var(--border)', paddingTop: '12px', marginTop: '4px' }}>
            <button
              style={{ width: '100%', background: 'transparent', border: '1px solid var(--border)', borderRadius: '7px', padding: '7px', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '500', cursor: 'pointer' }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = 'rgba(255,80,80,0.4)'; e.currentTarget.style.color = '#ff6b6b' }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--border)'; e.currentTarget.style.color = 'var(--text-muted)' }}
              onClick={logout}
            >
              Sign out
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
