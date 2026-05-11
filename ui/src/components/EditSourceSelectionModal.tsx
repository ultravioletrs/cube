import { useEffect, useMemo, useState } from 'react'
import { browseGoogleDrive, type DriveFileOption, type DriveFolderOption } from '@/lib/embedder/service'
import { resolveDriveFileTypeVisual } from '@/lib/embedder/file-type'
import type { DriveSource } from '@/types'

interface Props {
  authToken: string
  source: DriveSource
  onClose: () => void
  onSave: (selectedFileIDs: string[]) => Promise<void>
}

type PickerBrowseTab = 'folders' | 'shared_drives' | 'recent' | 'upload'
const PICKER_PAGE_SIZE = 80

export default function EditSourceSelectionModal({ authToken, source, onClose, onSave }: Props) {
  const [folderStack, setFolderStack] = useState<Array<{ id: string; name: string }>>([])
  const [currentFolderID, setCurrentFolderID] = useState(source.folderLink ?? '')
  const [folders, setFolders] = useState<DriveFolderOption[]>([])
  const [files, setFiles] = useState<DriveFileOption[]>([])
  const [selectedFileIDs, setSelectedFileIDs] = useState<string[]>(source.selectedFileIDs ?? [])
  const [fileSearch, setFileSearch] = useState('')
  const [pickerBrowseTab, setPickerBrowseTab] = useState<PickerBrowseTab>('folders')
  const [pickerSharedDriveID, setPickerSharedDriveID] = useState('')
  const [pickerUploadFile, setPickerUploadFile] = useState<File | null>(null)
  const [pickerUploading, setPickerUploading] = useState(false)
  const [pickerUploadProgress, setPickerUploadProgress] = useState(0)
  const [folderVisibleLimit, setFolderVisibleLimit] = useState(PICKER_PAGE_SIZE)
  const [fileVisibleLimit, setFileVisibleLimit] = useState(PICKER_PAGE_SIZE)
  const [fileViewMode, setFileViewMode] = useState<'cards' | 'list'>('cards')
  const [filesLoading, setFilesLoading] = useState(false)
  const [filesError, setFilesError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')

  const visibleFiles = useMemo(() => {
    const q = fileSearch.trim().toLowerCase()
    if (!q) return files
    return files.filter(file => file.name.toLowerCase().includes(q))
  }, [files, fileSearch])
  const shownFolders = useMemo(() => folders.slice(0, folderVisibleLimit), [folders, folderVisibleLimit])
  const shownFiles = useMemo(() => visibleFiles.slice(0, fileVisibleLimit), [visibleFiles, fileVisibleLimit])

  const folderPathItems = useMemo(() => {
    if (pickerBrowseTab === 'recent') {
      return [{ id: '', name: 'Recent' }]
    }
    if (pickerBrowseTab === 'shared_drives') {
      if (folderStack.length === 0) return [{ id: '', name: 'Shared drives' }]
      return [{ id: '', name: 'Shared drives' }, ...folderStack]
    }
    return [{ id: '', name: 'My Drive' }, ...folderStack]
  }, [folderStack, pickerBrowseTab])

  async function loadFolder(
    folderID: string,
    nextStack: Array<{ id: string; name: string }>,
    browseView: PickerBrowseTab = pickerBrowseTab,
    sharedDriveID: string = pickerSharedDriveID,
  ) {
    if (!source.accessToken) {
      setFilesError('This source is missing Google access token. Reconnect source credentials first.')
      return
    }

    setFilesLoading(true)
    setFilesError('')
    try {
      const result = await browseGoogleDrive(authToken, {
        accessToken: source.accessToken,
        refreshToken: source.refreshToken,
        browseFolderID: folderID,
        browseView: browseView === 'upload' ? 'folders' : browseView,
        sharedDriveID,
        folderID: source.folderLink,
      })
      setCurrentFolderID(result.currentFolderID ?? folderID)
      setPickerSharedDriveID(result.currentSharedDriveID ?? sharedDriveID)
      setFolders(result.folders)
      setFiles(result.files)
      if (browseView === 'recent') {
        setFolderStack([])
      } else {
        setFolderStack(nextStack)
      }
      setFolderVisibleLimit(PICKER_PAGE_SIZE)
      setFileVisibleLimit(PICKER_PAGE_SIZE)
      if (result.folders.length === 0 && result.files.length === 0) {
        setFilesError('No files or folders found in this location.')
      }
    } catch (err) {
      setFilesError(err instanceof Error ? err.message : 'Failed to load Google Drive files')
      setFolders([])
      setFiles([])
    } finally {
      setFilesLoading(false)
    }
  }

  useEffect(() => {
    void loadFolder(source.folderLink ?? '', [], 'folders', '')
  }, [source.folderLink])

  function toggleFile(fileID: string, checked: boolean) {
    setSelectedFileIDs(prev => {
      if (checked) return Array.from(new Set([...prev, fileID]))
      return prev.filter(id => id !== fileID)
    })
  }

  function navigateToBreadcrumb(index: number) {
    if (index === 0) {
      if (pickerBrowseTab === 'shared_drives') {
        void loadFolder('', [], 'shared_drives', '')
      } else {
        void loadFolder('', [], pickerBrowseTab === 'upload' ? 'folders' : pickerBrowseTab, pickerSharedDriveID)
      }
      return
    }
    if (pickerBrowseTab === 'shared_drives') {
      const drive = folderStack[0]
      if (!drive) {
        void loadFolder('', [], 'shared_drives', '')
        return
      }
      if (index === 1) {
        void loadFolder('', [drive], 'shared_drives', drive.id)
        return
      }
      const nextStack = folderStack.slice(0, index)
      const folder = nextStack[nextStack.length - 1]
      const folderID = folder ? folder.id : ''
      void loadFolder(folderID, nextStack, 'shared_drives', drive.id)
      return
    }
    const nextStack = folderStack.slice(0, index)
    const folder = nextStack[nextStack.length - 1]
    const folderID = folder ? folder.id : ''
    void loadFolder(folderID, nextStack, pickerBrowseTab === 'upload' ? 'folders' : pickerBrowseTab, pickerSharedDriveID)
  }

  function switchPickerTab(tab: PickerBrowseTab) {
    setPickerBrowseTab(tab)
    setFilesError('')
    setFileSearch('')
    setFolderVisibleLimit(PICKER_PAGE_SIZE)
    setFileVisibleLimit(PICKER_PAGE_SIZE)
    if (tab !== 'upload') {
      setPickerUploadFile(null)
      setPickerUploadProgress(0)
    }

    if (tab === 'recent') {
      setPickerSharedDriveID('')
      setFolderStack([])
      void loadFolder('', [], 'recent', '')
      return
    }
    if (tab === 'shared_drives') {
      setPickerSharedDriveID('')
      setFolderStack([])
      void loadFolder('', [], 'shared_drives', '')
      return
    }
    if (tab === 'folders') {
      setPickerSharedDriveID('')
      setFolderStack([])
      void loadFolder(source.folderLink ?? '', [], 'folders', '')
      return
    }
  }

  async function uploadPickerFile() {
    if (!pickerUploadFile) {
      setFilesError('Choose a file first.')
      return
    }
    if (!source.accessToken) {
      setFilesError('This source is missing Google access token. Reconnect source credentials first.')
      return
    }

    setPickerUploading(true)
    setPickerUploadProgress(5)
    setFilesError('')
    try {
      const buffer = await pickerUploadFile.arrayBuffer()
      setPickerUploadProgress(35)
      const bytes = new Uint8Array(buffer)
      let binary = ''
      for (let idx = 0; idx < bytes.length; idx += 1) {
        binary += String.fromCharCode(bytes[idx])
      }
      setPickerUploadProgress(65)
      const result = await browseGoogleDrive(authToken, {
        accessToken: source.accessToken,
        refreshToken: source.refreshToken,
        browseView: 'upload',
        browseFolderID: currentFolderID,
        sharedDriveID: pickerSharedDriveID,
        uploadName: pickerUploadFile.name,
        uploadMimeType: pickerUploadFile.type || 'application/octet-stream',
        uploadContentBase64: btoa(binary),
        folderID: source.folderLink,
      })
      setPickerUploadProgress(90)
      setPickerUploadFile(null)
      if (result.files.length > 0) {
        setSelectedFileIDs(prev => Array.from(new Set([...prev, result.files[0].id])))
      }
      if (pickerSharedDriveID) {
        await loadFolder(currentFolderID, folderStack, 'shared_drives', pickerSharedDriveID)
        setPickerBrowseTab('shared_drives')
      } else {
        await loadFolder(currentFolderID, folderStack, 'folders', '')
        setPickerBrowseTab('folders')
      }
      setPickerUploadProgress(100)
    } catch (err) {
      setFilesError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setPickerUploading(false)
      setTimeout(() => setPickerUploadProgress(0), 500)
    }
  }

  async function handleSave() {
    setSaveError('')
    setSaving(true)
    try {
      await onSave(selectedFileIDs)
      onClose()
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save source selection')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.35)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 105,
        padding: '16px',
        boxSizing: 'border-box',
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: '100%',
          maxWidth: '1060px',
          height: 'min(82vh, 760px)',
          background: 'var(--card-bg)',
          border: '1px solid var(--border)',
          borderRadius: '10px',
          boxShadow: '0 14px 40px rgba(0,0,0,0.35)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', gap: '10px', background: 'rgba(255,255,255,0.03)' }}>
          <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: 700, fontSize: '15px', color: 'var(--text)' }}>
            Source File Picker
          </div>
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
            Google Drive
          </div>
          <div style={{ marginLeft: '14px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '220px' }}>
            {source.name}
          </div>
          <button
            type="button"
            onClick={onClose}
            style={{ marginLeft: 'auto', width: '30px', height: '30px', border: '1px solid var(--border)', background: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: '18px', lineHeight: 1, padding: 0, borderRadius: '6px' }}
            aria-label="Close picker"
          >
            x
          </button>
        </div>

        <div style={{ padding: '0 16px', borderBottom: '1px solid var(--border)', background: 'var(--card-bg)', display: 'flex', alignItems: 'center', gap: '16px' }}>
          {([
            { id: 'folders', label: 'Folders' },
            { id: 'shared_drives', label: 'Shared drives' },
            { id: 'recent', label: 'Recent' },
            { id: 'upload', label: 'Upload' },
          ] as Array<{ id: PickerBrowseTab; label: string }>).map(tab => (
            <button
              key={tab.id}
              type="button"
              onClick={() => switchPickerTab(tab.id)}
              style={{
                border: 'none',
                background: 'none',
                borderBottom: pickerBrowseTab === tab.id ? '2px solid var(--accent)' : '2px solid transparent',
                color: pickerBrowseTab === tab.id ? 'var(--text)' : 'var(--text-dim)',
                padding: '10px 2px 9px',
                fontFamily: 'Space Grotesk, sans-serif',
                fontWeight: pickerBrowseTab === tab.id ? 600 : 500,
                fontSize: '12px',
                cursor: 'pointer',
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <div style={{ display: 'flex', minWidth: 0, flex: 1, alignItems: 'center', border: '1px solid var(--border)', background: 'rgba(255,255,255,0.03)', height: '30px', borderRadius: '6px' }}>
            <button type="button" style={{ border: 'none', borderRadius: '6px 0 0 6px', background: 'rgba(0,212,180,0.12)', color: 'var(--text)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', height: '100%', padding: '0 8px', cursor: 'default' }}>
              {pickerBrowseTab === 'shared_drives' ? 'Shared drives' : pickerBrowseTab === 'recent' ? 'Recent' : pickerBrowseTab === 'upload' ? 'Upload' : 'Folders'}
            </button>
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', padding: '0 8px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {folderPathItems.map(item => item.name).join(' / ')}
            </div>
          </div>
          <button
            type="button"
            onClick={() => {
              const view = pickerBrowseTab === 'upload' ? 'folders' : pickerBrowseTab
              void loadFolder(currentFolderID, folderStack, view, pickerSharedDriveID)
            }}
            disabled={filesLoading || pickerBrowseTab === 'upload'}
            style={{ minWidth: '90px', height: '30px', border: '1px solid var(--border)', borderRadius: '6px', background: 'rgba(255,255,255,0.03)', color: 'var(--text-muted)', cursor: filesLoading || pickerBrowseTab === 'upload' ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', opacity: filesLoading || pickerBrowseTab === 'upload' ? 0.6 : 1 }}
          >
            {filesLoading ? 'Refreshing…' : 'Refresh'}
          </button>
          <input
            type="text"
            placeholder="Search"
            value={fileSearch}
            onChange={e => setFileSearch(e.target.value)}
            disabled={pickerBrowseTab === 'upload'}
            style={{ width: '220px', height: '30px', border: '1px solid var(--border)', borderRadius: '6px', background: 'rgba(255,255,255,0.03)', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', padding: '0 10px', outline: 'none' }}
          />
          <div style={{ display: 'inline-flex', alignItems: 'center', border: '1px solid var(--border)', borderRadius: '6px', overflow: 'hidden' }}>
            <button
              type="button"
              onClick={() => setFileViewMode('cards')}
              style={{ height: '30px', minWidth: '56px', border: 'none', borderRight: '1px solid var(--border)', background: fileViewMode === 'cards' ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.03)', color: fileViewMode === 'cards' ? 'var(--text)' : 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
            >
              Cards
            </button>
            <button
              type="button"
              onClick={() => setFileViewMode('list')}
              style={{ height: '30px', minWidth: '56px', border: 'none', background: fileViewMode === 'list' ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.03)', color: fileViewMode === 'list' ? 'var(--text)' : 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
            >
              List
            </button>
          </div>
        </div>

        <div style={{ padding: '8px 16px', borderBottom: '1px solid var(--border)', background: 'rgba(255,255,255,0.01)', display: 'flex', alignItems: 'center', gap: '6px', overflowX: 'auto' }}>
          {folderPathItems.map((item, index) => (
            <div key={`${item.id || 'root'}-${index}`} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              {index > 0 && <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>/</span>}
              <button
                type="button"
                onClick={() => navigateToBreadcrumb(index)}
                style={{ border: '1px solid var(--border)', borderRadius: '6px', background: index === folderPathItems.length - 1 ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.03)', color: 'var(--text)', padding: '4px 8px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer', whiteSpace: 'nowrap' }}
              >
                {item.name}
              </button>
            </div>
          ))}
        </div>

        <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', background: 'var(--card-bg)', padding: '12px 16px 16px' }}>
          {folders.length > 0 && (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(175px, 1fr))', gap: '10px', marginBottom: '14px' }}>
              {shownFolders.map(folder => (
                <button
                  key={folder.id}
                  type="button"
                  onClick={() => {
                    if (pickerBrowseTab === 'shared_drives' && !pickerSharedDriveID && folder.mimeType === 'application/vnd.google-apps.drive') {
                      const driveStack = [{ id: folder.id, name: folder.name }]
                      void loadFolder('', driveStack, 'shared_drives', folder.id)
                      return
                    }
                    const next = [...folderStack, { id: folder.id, name: folder.name }]
                    void loadFolder(folder.id, next, pickerBrowseTab === 'upload' ? 'folders' : pickerBrowseTab, pickerSharedDriveID)
                  }}
                  style={{ border: '1px solid var(--border)', borderRadius: '8px', background: 'rgba(255,255,255,0.03)', height: '36px', display: 'flex', alignItems: 'center', gap: '8px', padding: '0 10px', cursor: 'pointer', color: 'var(--text)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px' }}
                >
                  <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>[DIR]</span>
                  <span style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {folder.name}
                  </span>
                </button>
              ))}
              {folders.length > shownFolders.length && (
                <button
                  type="button"
                  onClick={() => setFolderVisibleLimit(prev => prev + PICKER_PAGE_SIZE)}
                  style={{ border: '1px dashed var(--border)', borderRadius: '8px', background: 'rgba(255,255,255,0.02)', height: '36px', color: 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                >
                  Show more folders
                </button>
              )}
            </div>
          )}

          {pickerBrowseTab === 'upload' && (
            <div style={{ border: '1px dashed var(--border)', borderRadius: '10px', padding: '12px', marginBottom: '14px', background: 'rgba(255,255,255,0.02)' }}>
              <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', marginBottom: '8px', fontWeight: 600 }}>
                Upload file to {folderPathItems.map(item => item.name).join(' / ')}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <input
                  type="file"
                  onChange={e => {
                    const target = e.currentTarget.files?.[0] ?? null
                    setPickerUploadFile(target)
                  }}
                  style={{ flex: 1, fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}
                />
                <button
                  type="button"
                  onClick={() => { void uploadPickerFile() }}
                  disabled={pickerUploading || !pickerUploadFile}
                  style={{ minWidth: '92px', height: '30px', border: 'none', borderRadius: '6px', background: 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 700, cursor: pickerUploading || !pickerUploadFile ? 'default' : 'pointer', opacity: pickerUploading || !pickerUploadFile ? 0.7 : 1 }}
                >
                  {pickerUploading ? 'Uploading…' : 'Upload'}
                </button>
              </div>
              {pickerUploading && (
                <div style={{ marginTop: '8px' }}>
                  <div style={{ height: '6px', borderRadius: '999px', background: 'rgba(255,255,255,0.08)', overflow: 'hidden' }}>
                    <div style={{ height: '100%', width: `${pickerUploadProgress}%`, background: 'var(--accent)', transition: 'width 160ms ease' }} />
                  </div>
                  <div style={{ marginTop: '4px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>
                    {pickerUploadProgress}% uploaded
                  </div>
                </div>
              )}
            </div>
          )}

          {fileViewMode === 'cards' ? (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(185px, 1fr))', gap: '12px' }}>
              {shownFiles.map(file => {
                const selected = selectedFileIDs.includes(file.id)
                const fileVisual = resolveDriveFileTypeVisual(file)
                const mimeTag = `[${fileVisual.badge}]`
                const iconURL = (file.iconLink || '').trim()

                return (
                  <label
                    key={file.id}
                    style={{
                      border: selected ? '2px solid var(--accent)' : '1px solid var(--border)',
                      borderRadius: '8px',
                      background: 'rgba(255,255,255,0.03)',
                      display: 'flex',
                      flexDirection: 'column',
                      cursor: 'pointer',
                      minHeight: '188px',
                      position: 'relative',
                      padding: '0',
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={selected}
                      onChange={e => {
                        toggleFile(file.id, e.target.checked)
                      }}
                      style={{ position: 'absolute', top: '8px', left: '8px', zIndex: 1, accentColor: 'var(--accent)' }}
                    />
                    <div style={{ height: '140px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(255,255,255,0.02)', overflow: 'hidden' }}>
                      <div style={{ width: '78px', height: '96px', borderRadius: '10px', border: `1px solid ${fileVisual.iconBorder}`, background: fileVisual.iconBg, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '7px' }}>
                        {iconURL ? (
                          <img
                            src={iconURL}
                            alt={`${fileVisual.label} icon`}
                            loading="lazy"
                            referrerPolicy="no-referrer"
                            style={{ width: '26px', height: '26px', objectFit: 'contain', opacity: 0.95 }}
                          />
                        ) : (
                          <div style={{ width: '26px', height: '26px', borderRadius: '6px', border: `1px solid ${fileVisual.iconBorder}`, color: fileVisual.iconColor, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', fontWeight: 700 }}>
                            {fileVisual.badge.slice(0, 2)}
                          </div>
                        )}
                        <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: fileVisual.iconColor, fontWeight: 700, letterSpacing: '0.05em' }}>
                          {fileVisual.badge}
                        </div>
                      </div>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '7px', padding: '8px 9px 9px' }}>
                      <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{mimeTag}</span>
                      <div style={{ minWidth: 0, flex: 1 }}>
                        <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {file.name}
                        </div>
                      </div>
                    </div>
                  </label>
                )
              })}
            </div>
          ) : (
            <div style={{ border: '1px solid var(--border)', borderRadius: '8px', overflow: 'hidden' }}>
              {shownFiles.map(file => {
                const selected = selectedFileIDs.includes(file.id)
                const fileVisual = resolveDriveFileTypeVisual(file)
                const mimeTag = `[${fileVisual.badge}]`
                const iconURL = (file.iconLink || '').trim()

                return (
                  <label
                    key={file.id}
                    style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '9px 10px', borderBottom: '1px solid var(--border)', background: selected ? 'rgba(0,212,180,0.08)' : 'rgba(255,255,255,0.02)', cursor: 'pointer' }}
                  >
                    <input
                      type="checkbox"
                      checked={selected}
                      onChange={e => {
                        toggleFile(file.id, e.target.checked)
                      }}
                      style={{ accentColor: 'var(--accent)' }}
                    />
                    <div style={{ width: '26px', height: '26px', borderRadius: '6px', border: `1px solid ${fileVisual.iconBorder}`, background: fileVisual.iconBg, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                      {iconURL ? (
                        <img
                          src={iconURL}
                          alt={`${fileVisual.label} icon`}
                          loading="lazy"
                          referrerPolicy="no-referrer"
                          style={{ width: '16px', height: '16px', objectFit: 'contain' }}
                        />
                      ) : (
                        <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '9px', color: fileVisual.iconColor, fontWeight: 700 }}>{fileVisual.badge.slice(0, 2)}</span>
                      )}
                    </div>
                    <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', minWidth: '50px' }}>{mimeTag}</span>
                    <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', minWidth: 0, flex: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{file.name}</span>
                  </label>
                )
              })}
            </div>
          )}

          {visibleFiles.length > shownFiles.length && (
            <div style={{ marginTop: '10px', display: 'flex', justifyContent: 'center' }}>
              <button
                type="button"
                onClick={() => setFileVisibleLimit(prev => prev + PICKER_PAGE_SIZE)}
                style={{ border: '1px dashed var(--border)', borderRadius: '8px', background: 'rgba(255,255,255,0.02)', color: 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', padding: '7px 11px', cursor: 'pointer' }}
              >
                Show more files
              </button>
            </div>
          )}

          {folders.length === 0 && visibleFiles.length === 0 && (
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)', marginTop: '4px' }}>
              No items in this folder.
            </div>
          )}
        </div>

        <div style={{ padding: '10px 16px', borderTop: '1px solid var(--border)', background: 'rgba(255,255,255,0.02)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text-dim)' }}>
            {selectedFileIDs.length} files marked
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <button
              type="button"
              onClick={() => {
                void handleSave()
              }}
              disabled={saving}
              style={{ minWidth: '82px', height: '30px', border: 'none', borderRadius: '6px', background: 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 700, cursor: saving ? 'default' : 'pointer', opacity: saving ? 0.7 : 1 }}
            >
              {saving ? 'Saving...' : 'Select'}
            </button>
            <button
              type="button"
              onClick={onClose}
              style={{ minWidth: '82px', height: '30px', border: '1px solid var(--border)', borderRadius: '6px', background: 'rgba(255,255,255,0.03)', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}
            >
              Cancel
            </button>
          </div>
        </div>

        {(filesError || saveError) && (
          <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#b42318', background: '#fef3f2', padding: '10px 16px', borderTop: '1px solid #fecdca' }}>
            {filesError || saveError}
          </div>
        )}
      </div>
    </div>
  )
}
