import { useNavigate, useLocation } from 'react-router-dom'

const navItems = [
  {
    id: 'records',
    label: 'Records',
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <rect x="3" y="2" width="10" height="13" rx="1.5" stroke="currentColor" strokeWidth="1.5"/>
        <path d="M6 6h4M6 9h4M6 12h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
        <rect x="7" y="5" width="10" height="13" rx="1.5" stroke="currentColor" strokeWidth="1.5" fill="var(--bg)"/>
        <path d="M10 9h4M10 12h4M10 15h2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      </svg>
    ),
  },
  {
    id: 'sources',
    label: 'Sources',
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <ellipse cx="10" cy="5.5" rx="7" ry="2.5" stroke="currentColor" strokeWidth="1.5"/>
        <path d="M3 5.5v9c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5v-9" stroke="currentColor" strokeWidth="1.5"/>
        <path d="M3 10c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5" stroke="currentColor" strokeWidth="1.5"/>
      </svg>
    ),
  },
  {
    id: 'chat',
    label: 'Prompt',
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path d="M3 4.5A1.5 1.5 0 014.5 3h11A1.5 1.5 0 0117 4.5v8A1.5 1.5 0 0115.5 14H11l-3 3v-3H4.5A1.5 1.5 0 013 12.5v-8z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round"/>
        <path d="M7 8h6M7 11h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      </svg>
    ),
  },
  {
    id: 'config',
    label: 'Config',
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="2.5" stroke="currentColor" strokeWidth="1.5"/>
        <path d="M10 3v2M10 15v2M3 10h2M15 10h2M4.93 4.93l1.41 1.41M13.66 13.66l1.41 1.41M4.93 15.07l1.41-1.41M13.66 6.34l1.41-1.41" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      </svg>
    ),
  },
]

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  return (
    <aside style={{ width: '220px', minWidth: '220px', height: '100%', background: 'var(--sidebar-bg)', borderRight: '1px solid var(--border)', display: 'flex', flexDirection: 'column', position: 'relative', zIndex: 10 }}>
      <button onClick={() => navigate('/records')} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '24px 20px', borderBottom: '1px solid var(--border)', marginBottom: '8px', background: 'none', border: 'none', cursor: 'pointer', width: '100%', textAlign: 'left' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
            <path d="M4 4h6v6H4zM12 4h6v6h-6zM4 12h6v6H4z" fill="var(--accent)" opacity="0.9"/>
            <path d="M12 12h6v6h-6z" fill="var(--accent)" opacity="0.3"/>
          </svg>
        </div>
        <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '18px', lineHeight: 1, color: 'var(--text)', letterSpacing: '-0.02em' }}>Veda</span>
      </button>

      <nav style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '2px', padding: '8px 12px' }}>
        {navItems.map(item => {
          const active = location.pathname === '/' + item.id
          return (
            <button
              key={item.id}
              onClick={() => navigate('/' + item.id)}
              style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '9px 12px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontWeight: '500', fontSize: '14px', transition: 'all 0.15s ease', textAlign: 'left', width: '100%', background: active ? 'rgba(0,212,180,0.08)' : 'transparent', color: active ? 'var(--accent)' : 'var(--text-muted)', borderLeft: active ? '2px solid var(--accent)' : '2px solid transparent' }}
              onMouseEnter={e => { if (!active) { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,255,255,0.04)'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text)' } }}
              onMouseLeave={e => { if (!active) { (e.currentTarget as HTMLButtonElement).style.background = 'transparent'; (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)' } }}
            >
              <span style={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>{item.icon}</span>
              <span style={{ flex: 1 }}>{item.label}</span>
            </button>
          )
        })}
      </nav>
    </aside>
  )
}
