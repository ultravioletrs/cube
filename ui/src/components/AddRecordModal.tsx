// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useState } from 'react'

interface Props {
  onClose: () => void
  onUploadFile: (file: File) => Promise<void>
}

type RecordType = 'document' | 'image' | 'link'

const RECORD_TYPES: { value: RecordType; label: string; description: string; accept: string }[] = [
  { value: 'document', label: 'Document', description: 'PDF, TXT, Markdown, or DOCX', accept: '.pdf,.txt,.md,.docx' },
  { value: 'image',    label: 'Image',    description: 'PNG, JPG, WEBP, or GIF',  accept: '.png,.jpg,.jpeg,.webp,.gif' },
  { value: 'link',     label: 'Link',     description: 'Index any web page by URL', accept: '' },
]

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontFamily: 'Space Grotesk, sans-serif',
  fontWeight: '600',
  fontSize: '12px',
  color: 'var(--text)',
  marginBottom: '5px',
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  background: 'rgba(255,255,255,0.04)',
  border: '1px solid var(--border)',
  borderRadius: '8px',
  padding: '8px 11px',
  color: 'var(--text)',
  fontFamily: 'Space Grotesk, sans-serif',
  fontSize: '13px',
  outline: 'none',
  boxSizing: 'border-box',
}

const hintStyle: React.CSSProperties = {
  fontFamily: 'JetBrains Mono, monospace',
  fontSize: '10px',
  color: 'var(--text-dim)',
  marginTop: '4px',
}

const errorStyle: React.CSSProperties = {
  fontFamily: 'JetBrains Mono, monospace',
  fontSize: '10px',
  color: '#ff6b6b',
  marginTop: '4px',
}

function DocumentIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
      <rect x="3" y="1.5" width="11" height="15" rx="1.5" stroke="var(--text)" strokeWidth="1.4" opacity="0.7"/>
      <path d="M13 1.5v4h3.5" stroke="var(--text)" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" opacity="0.5"/>
      <path d="M6 8h6M6 11h4" stroke="var(--text)" strokeWidth="1.3" strokeLinecap="round" opacity="0.5"/>
    </svg>
  )
}

function ImageIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
      <rect x="2" y="3" width="16" height="14" rx="2" stroke="var(--text)" strokeWidth="1.4" opacity="0.7"/>
      <circle cx="7" cy="8" r="1.5" fill="var(--text)" opacity="0.5"/>
      <path d="M2 14l4-4 3 3 3-4 4 5" stroke="var(--text)" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" opacity="0.5"/>
    </svg>
  )
}

function LinkIconSvg() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
      <path d="M8.5 11.5a4.5 4.5 0 006.364 0l2-2a4.5 4.5 0 00-6.364-6.364l-1.5 1.5" stroke="var(--text)" strokeWidth="1.4" strokeLinecap="round" opacity="0.7"/>
      <path d="M11.5 8.5a4.5 4.5 0 00-6.364 0l-2 2a4.5 4.5 0 006.364 6.364l1.5-1.5" stroke="var(--text)" strokeWidth="1.4" strokeLinecap="round" opacity="0.7"/>
    </svg>
  )
}

function typeIcon(t: RecordType) {
  if (t === 'document') return <DocumentIcon />
  if (t === 'image') return <ImageIcon />
  return <LinkIconSvg />
}

function iconBg(t: RecordType) {
  if (t === 'document') return { bg: 'rgba(77,158,247,0.15)', border: 'rgba(77,158,247,0.25)' }
  if (t === 'image') return { bg: 'rgba(167,139,250,0.15)', border: 'rgba(167,139,250,0.25)' }
  return { bg: 'rgba(0,212,180,0.15)', border: 'rgba(0,212,180,0.25)' }
}

export default function AddRecordModal({ onClose, onUploadFile }: Props) {
  const [step, setStep] = useState<'select-type' | 'configure'>('select-type')
  const [recordType, setRecordType] = useState<RecordType>('document')

  const [files, setFiles] = useState<File[]>([])
  const [dragging, setDragging] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<{ done: number; total: number } | null>(null)

  const [url, setUrl] = useState('')
  const [linkName, setLinkName] = useState('')
  const [urlError, setUrlError] = useState('')
  const [formError, setFormError] = useState('')

  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  const selectedType = RECORD_TYPES.find(t => t.value === recordType)!

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragging(false)
    const dropped = Array.from(e.dataTransfer.files)
    if (dropped.length > 0) setFiles(prev => [...prev, ...dropped])
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files ?? [])
    if (selected.length > 0) setFiles(prev => [...prev, ...selected])
    e.target.value = ''
  }

  function removeFile(index: number) {
    setFiles(prev => prev.filter((_, i) => i !== index))
  }

  async function handleSave() {
    setFormError('')
    setSaved(false)
    if (recordType === 'link') {
      if (!url.trim()) { setUrlError('Required'); return }
      try { new URL(url.trim()) } catch { setUrlError('Enter a valid URL'); return }
    } else {
      if (files.length === 0) return
    }

    setSaving(true)
    try {
      if (recordType === 'link') {
        throw new Error('Direct link record creation is not supported yet. Add a source and run sync to create records.')
      }

      setUploadProgress({ done: 0, total: files.length })
      for (let i = 0; i < files.length; i++) {
        await onUploadFile(files[i])
        setUploadProgress({ done: i + 1, total: files.length })
      }
      setUploadProgress(null)
      setSaving(false)
      setSaved(true)
      setTimeout(() => onClose(), 700)
    } catch (err) {
      setUploadProgress(null)
      setSaving(false)
      setFormError(err instanceof Error ? err.message : 'Failed to upload record')
      console.error('record upload failed', err)
    }
  }

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(7,12,22,0.85)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100, backdropFilter: 'blur(4px)' }}
      onClick={onClose}
    >
      <div
        style={{ background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '16px', width: '480px', maxHeight: '90vh', display: 'flex', flexDirection: 'column', boxShadow: '0 24px 80px rgba(0,0,0,0.6)' }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{ padding: '22px 24px 18px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', gap: '10px', flexShrink: 0 }}>
          {step === 'configure' && (
            <button onClick={() => { setStep('select-type'); setFiles([]) }} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px', display: 'flex', alignItems: 'center', borderRadius: '6px' }}>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M10 3L5 8l5 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
            </button>
          )}
          <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '16px', color: 'var(--text)', flex: 1 }}>
            {step === 'select-type' ? 'Add Record' : selectedType.label}
          </span>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px', display: 'flex', borderRadius: '6px' }}>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/></svg>
          </button>
        </div>

        {/* Body */}
        <div style={{ overflowY: 'auto', flex: 1, padding: '22px 24px' }}>

          {step === 'select-type' ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', margin: '0 0 6px' }}>
                What would you like to add?
              </p>
              {RECORD_TYPES.map(t => {
                const colors = iconBg(t.value)
                return (
                  <button
                    key={t.value}
                    onClick={() => { setRecordType(t.value); setStep('configure') }}
                    style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '14px 16px', background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border)', borderRadius: '10px', cursor: 'pointer', textAlign: 'left', width: '100%', transition: 'all 0.15s ease' }}
                  >
                    <div style={{ width: '38px', height: '38px', borderRadius: '9px', background: colors.bg, border: `1px solid ${colors.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                      {typeIcon(t.value)}
                    </div>
                    <div>
                      <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>{t.label}</div>
                      <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', marginTop: '2px' }}>{t.description}</div>
                    </div>
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ marginLeft: 'auto', flexShrink: 0, opacity: 0.4 }}><path d="M5 3l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
                  </button>
                )
              })}
            </div>
          ) : recordType === 'link' ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={labelStyle}>URL *</label>
                <input
                  type="url"
                  placeholder="https://example.com/article"
                  value={url}
                  autoFocus
                  onChange={e => { setUrl(e.target.value); if (urlError) setUrlError('') }}
                  style={{ ...inputStyle, borderColor: urlError ? 'rgba(255,107,107,0.5)' : undefined }}
                />
                {urlError && <p style={errorStyle}>{urlError}</p>}
              </div>
              <div>
                <label style={labelStyle}>Name (optional)</label>
                <input
                  type="text"
                  placeholder="Defaults to the page title or domain"
                  value={linkName}
                  onChange={e => setLinkName(e.target.value)}
                  style={inputStyle}
                />
                <p style={hintStyle}>Leave empty to use the page title.</p>
              </div>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div
                onDragOver={e => { e.preventDefault(); setDragging(true) }}
                onDragLeave={() => setDragging(false)}
                onDrop={handleDrop}
                onClick={() => (document.getElementById('record-file-input') as HTMLInputElement)?.click()}
                style={{ border: `1.5px dashed ${dragging || files.length > 0 ? 'var(--accent)' : 'var(--border)'}`, borderRadius: '12px', padding: '28px 24px', cursor: 'pointer', transition: 'all 0.2s ease', display: 'flex', alignItems: 'center', justifyContent: 'center', background: dragging ? 'rgba(0,212,180,0.06)' : 'rgba(255,255,255,0.02)' }}
              >
                <input
                  id="record-file-input"
                  type="file"
                  accept={selectedType.accept}
                  multiple
                  style={{ display: 'none' }}
                  onChange={handleFileChange}
                />
                <div style={{ textAlign: 'center' }}>
                  <svg width="36" height="36" viewBox="0 0 36 36" fill="none" style={{ margin: '0 auto 10px', display: 'block', opacity: 0.35 }}>
                    <path d="M18 26V14M12 18l6-6 6 6" stroke="var(--text)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                    <rect x="5" y="5" width="26" height="26" rx="6" stroke="var(--text)" strokeWidth="1.5" opacity="0.4"/>
                  </svg>
                  <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', fontWeight: '500' }}>Drop files or click to browse</div>
                  <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', marginTop: '5px' }}>{selectedType.description}</div>
                </div>
              </div>
              {files.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', maxHeight: '180px', overflowY: 'auto' }}>
                  {files.map((f, i) => (
                    <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '10px', background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', borderRadius: '8px', padding: '7px 10px' }}>
                      <svg width="14" height="14" viewBox="0 0 20 20" fill="none" style={{ flexShrink: 0, opacity: 0.5 }}>
                        <rect x="3" y="1.5" width="11" height="15" rx="1.5" stroke="var(--text)" strokeWidth="1.4"/>
                        <path d="M6 8h6M6 11h4" stroke="var(--text)" strokeWidth="1.3" strokeLinecap="round"/>
                      </svg>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{f.name}</div>
                        <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{(f.size / 1024 / 1024).toFixed(2)} MB</div>
                      </div>
                      <button
                        type="button"
                        onClick={e => { e.stopPropagation(); removeFile(i) }}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-dim)', padding: '2px', display: 'flex', flexShrink: 0 }}
                        onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.color = '#ff5050' }}
                        onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-dim)' }}
                      >
                        <svg width="12" height="12" viewBox="0 0 14 14" fill="none"><path d="M4 4l6 6M10 4l-6 6" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/></svg>
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div style={{ padding: '16px 24px', borderTop: '1px solid var(--border)', display: 'flex', justifyContent: 'flex-end', gap: '10px', flexShrink: 0 }}>
          {formError && (
            <div style={{ marginRight: 'auto', ...errorStyle, marginTop: 0, maxWidth: '280px' }}>{formError}</div>
          )}
          <button
            onClick={onClose}
            style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', padding: '8px 18px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '500' }}
          >
            Cancel
          </button>
          {step === 'configure' && (
            <button
              onClick={handleSave}
              disabled={saving || saved || (recordType !== 'link' && files.length === 0)}
              style={{ background: saved ? 'rgba(0,212,180,0.7)' : 'var(--accent)', color: '#070c16', padding: '8px 20px', border: 'none', borderRadius: '8px', cursor: saving || saved || (recordType !== 'link' && files.length === 0) ? 'default' : 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', opacity: saving || (recordType !== 'link' && files.length === 0) ? 0.5 : 1, minWidth: '100px' }}
            >
              {saved ? '✓ Added' : uploadProgress ? `Uploading ${uploadProgress.done}/${uploadProgress.total}…` : recordType === 'link' ? 'Add Link' : files.length > 1 ? `Upload (${files.length})` : 'Upload'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
