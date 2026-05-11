import type { DriveSource } from '@/types'

interface Props {
  authToken: string
  source: DriveSource
  onClose: () => void
  onSaved: () => Promise<void>
}

export default function SourceScopeModal({ onClose }: Props) {
  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(7,12,22,0.85)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 100,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'var(--card-bg)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '18px 20px',
          width: '420px',
          maxWidth: 'calc(100vw - 24px)',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: 700, fontSize: '15px', color: 'var(--text)', marginBottom: '10px' }}>
          Scope management moved to source creation
        </div>
        <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', marginBottom: '14px' }}>
          Recreate the source to change selected files.
        </div>
        <button
          onClick={onClose}
          style={{ background: 'var(--accent)', border: 'none', color: '#070c16', padding: '8px 12px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: '700' }}
        >
          Close
        </button>
      </div>
    </div>
  )
}
