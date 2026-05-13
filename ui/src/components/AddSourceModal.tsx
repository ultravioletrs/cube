// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useEffect, useMemo, useState } from 'react'
import {
  browseGoogleDrive,
  browseRclonePath,
  exchangeGoogleOAuth,
  getGoogleOAuthURL,
  type DriveFileOption,
  type DriveFolderOption,
  type RcloneFolderOption,
} from '@/lib/embedder/service'
import { resolveDriveFileTypeVisual } from '@/lib/embedder/file-type'
import type { DriveSourceDraft } from '@/types'

interface Props {
  authToken: string
  onClose: () => void
  onAdd: (source: DriveSourceDraft) => Promise<void>
  initialGoogleAccessToken?: string
  initialGoogleRefreshToken?: string
}

type PickerBrowseTab = 'folders' | 'shared_drives' | 'recent' | 'upload'
const PICKER_PAGE_SIZE = 80

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

function Field({ label, hint, error, children }: { label: string; hint?: string; error?: string; children: React.ReactNode }) {
  return (
    <div>
      <label style={labelStyle}>{label}</label>
      {children}
      {hint && !error && <p style={hintStyle}>{hint}</p>}
      {error && <p style={errorStyle}>{error}</p>}
    </div>
  )
}

function GoogleDriveIcon({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <path d="M8.5 3H15.5L22 13H16L13 8.5L10 13H2L8.5 3Z" fill="#4285f4" opacity="0.85"/>
      <path d="M2 13L5.5 19H18.5L22 13H16L13 18H11L8 13H2Z" fill="#34a853" opacity="0.85"/>
      <path d="M10 13L13 8.5L16 13H10Z" fill="#fbbc04"/>
    </svg>
  )
}

export default function AddSourceModal({
  authToken,
  onClose,
  onAdd,
  initialGoogleAccessToken = '',
  initialGoogleRefreshToken = '',
}: Props) {
  const [providerTab, setProviderTab] = useState<'google' | 'rclone'>('google')
  const [name, setName] = useState('')
  const [importMode, setImportMode] = useState<'selected' | 'all'>('selected')

  const [oauthConnecting, setOauthConnecting] = useState(false)
  const [oauthConnected, setOauthConnected] = useState(Boolean(initialGoogleAccessToken))
  const [googleAccessToken, setGoogleAccessToken] = useState(initialGoogleAccessToken)
  const [googleRefreshToken, setGoogleRefreshToken] = useState(initialGoogleRefreshToken)

  const [folderStack, setFolderStack] = useState<Array<{ id: string; name: string }>>([])
  const [currentFolderID, setCurrentFolderID] = useState('')
  const [folders, setFolders] = useState<DriveFolderOption[]>([])
  const [files, setFiles] = useState<DriveFileOption[]>([])
  const [selectedFileIDs, setSelectedFileIDs] = useState<string[]>([])
  const [selectedFileMetaByID, setSelectedFileMetaByID] = useState<Record<string, DriveFileOption>>({})

  const [fileSearch, setFileSearch] = useState('')
  const [pickerBrowseTab, setPickerBrowseTab] = useState<PickerBrowseTab>('folders')
  const [pickerSharedDriveID, setPickerSharedDriveID] = useState('')
  const [pickerUploadFile, setPickerUploadFile] = useState<File | null>(null)
  const [pickerUploading, setPickerUploading] = useState(false)
  const [pickerUploadProgress, setPickerUploadProgress] = useState(0)
  const [folderVisibleLimit, setFolderVisibleLimit] = useState(PICKER_PAGE_SIZE)
  const [fileVisibleLimit, setFileVisibleLimit] = useState(PICKER_PAGE_SIZE)
  const [pickerViewMode, setPickerViewMode] = useState<'cards' | 'list'>('cards')
  const [filesLoading, setFilesLoading] = useState(false)
  const [filesLoaded, setFilesLoaded] = useState(false)
  const [filesError, setFilesError] = useState('')

  const [showFilePicker, setShowFilePicker] = useState(false)
  const [pickerSelectionIDs, setPickerSelectionIDs] = useState<string[]>([])

  const [rcloneRemote, setRcloneRemote] = useState('')
  const [rcloneConfigRef, setRcloneConfigRef] = useState('')
  const [rcloneCurrentPath, setRcloneCurrentPath] = useState('')
  const [rcloneParentPath, setRcloneParentPath] = useState<string | undefined>(undefined)
  const [rcloneFolders, setRcloneFolders] = useState<RcloneFolderOption[]>([])
  const [rcloneFiles, setRcloneFiles] = useState<DriveFileOption[]>([])
  const [rcloneSelection, setRcloneSelection] = useState<string[]>([])
  const [rcloneSelectionMeta, setRcloneSelectionMetaMap] = useState<Record<string, { name: string; path: string; kind: 'file' | 'folder' }>>({})
  const [rcloneSearch, setRcloneSearch] = useState('')
  const [rcloneLoading, setRcloneLoading] = useState(false)
  const [rcloneLoaded, setRcloneLoaded] = useState(false)
  const [rcloneError, setRcloneError] = useState('')

  const [syncEnabled, setSyncEnabled] = useState(true)
  const [autoSyncInterval, setAutoSyncInterval] = useState('60')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  const visibleFiles = useMemo(() => {
    const q = fileSearch.trim().toLowerCase()
    if (!q) return files
    return files.filter(file => file.name.toLowerCase().includes(q))
  }, [files, fileSearch])

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

  const folderPathLabel = useMemo(() => folderPathItems.map(item => item.name).join(' / '), [folderPathItems])

  const visibleRcloneFolders = useMemo(() => {
    const q = rcloneSearch.trim().toLowerCase()
    if (!q) return rcloneFolders
    return rcloneFolders.filter(folder => folder.name.toLowerCase().includes(q) || folder.path.toLowerCase().includes(q))
  }, [rcloneFolders, rcloneSearch])

  const visibleRcloneFiles = useMemo(() => {
    const q = rcloneSearch.trim().toLowerCase()
    if (!q) return rcloneFiles
    return rcloneFiles.filter(file => file.name.toLowerCase().includes(q) || file.path.toLowerCase().includes(q))
  }, [rcloneFiles, rcloneSearch])

  const shownFolders = useMemo(() => folders.slice(0, folderVisibleLimit), [folders, folderVisibleLimit])
  const shownFiles = useMemo(() => visibleFiles.slice(0, fileVisibleLimit), [visibleFiles, fileVisibleLimit])

  function clearFieldError(field: string) {
    setErrors(prev => {
      if (!prev[field]) return prev
      const next = { ...prev }
      delete next[field]
      return next
    })
  }

  function clearSelectionError() {
    clearFieldError('selectedFileIDs')
  }

  function setMetaForFiles(nextFiles: DriveFileOption[]) {
    if (nextFiles.length === 0) return
    setSelectedFileMetaByID(prev => {
      const next = { ...prev }
      for (const file of nextFiles) next[file.id] = file
      return next
    })
  }

  function removeSelectedFile(id: string) {
    setSelectedFileIDs(prev => prev.filter(fileID => fileID !== id))
    setSelectedFileMetaByID(prev => {
      if (!prev[id]) return prev
      const next = { ...prev }
      delete next[id]
      return next
    })
    clearSelectionError()
  }

  async function loadFolder(
    folderID: string,
    nextStack: Array<{ id: string; name: string }>,
    browseView: PickerBrowseTab = pickerBrowseTab,
    sharedDriveID: string = pickerSharedDriveID,
  ) {
    if (!googleAccessToken) {
      setFilesError('Connect Google Drive first.')
      return
    }

    setFilesLoading(true)
    setFilesError('')
    try {
      const result = await browseGoogleDrive(authToken, {
        accessToken: googleAccessToken,
        refreshToken: googleRefreshToken,
        browseFolderID: folderID,
        browseView: browseView === 'upload' ? 'folders' : browseView,
        sharedDriveID,
      })
      setCurrentFolderID(result.currentFolderID ?? folderID)
      setPickerSharedDriveID(result.currentSharedDriveID ?? sharedDriveID)
      setFolders(result.folders)
      setFiles(result.files)
      setMetaForFiles(result.files)
      if (browseView === 'recent') {
        setFolderStack([])
      } else {
        setFolderStack(nextStack)
      }
      setFolderVisibleLimit(PICKER_PAGE_SIZE)
      setFileVisibleLimit(PICKER_PAGE_SIZE)
      setFilesLoaded(true)
      if (result.folders.length === 0 && result.files.length === 0) {
        setFilesError('No files or folders found in this location.')
      }
    } catch (err) {
      setFilesError(err instanceof Error ? err.message : 'Failed to load Google Drive files')
      setFolders([])
      setFiles([])
      setFilesLoaded(false)
    } finally {
      setFilesLoading(false)
    }
  }

  async function connectGoogleDrive() {
    setFormError('')
    setFilesError('')
    setOauthConnecting(true)

    const redirectURI = `${window.location.origin}/oauth/google/callback`
    try {
      const { authURL } = await getGoogleOAuthURL(authToken, redirectURI)
      const popup = window.open(authURL, 'veda-google-oauth', 'width=520,height=700')
      if (!popup) {
        throw new Error('Popup blocked. Allow popups and try again.')
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to start Google OAuth')
      setOauthConnecting(false)
    }
  }

  useEffect(() => {
    if (!initialGoogleAccessToken) return
    setGoogleAccessToken(prev => prev || initialGoogleAccessToken)
    setGoogleRefreshToken(prev => prev || initialGoogleRefreshToken)
    setOauthConnected(true)
  }, [initialGoogleAccessToken, initialGoogleRefreshToken])

  useEffect(() => {
    async function handleMessage(event: MessageEvent) {
      if (event.origin !== window.location.origin) return
      const payload = event.data as { type?: string; code?: string; state?: string; error?: string }
      if (!payload || payload.type !== 'google_oauth_callback') return

      if (payload.error) {
        setFormError(`Google OAuth failed: ${payload.error}`)
        setOauthConnecting(false)
        return
      }
      if (!payload.code || !payload.state) {
        setFormError('Google OAuth callback is missing code/state.')
        setOauthConnecting(false)
        return
      }

      const redirectURI = `${window.location.origin}/oauth/google/callback`
      try {
        const tokens = await exchangeGoogleOAuth(authToken, payload.code, payload.state, redirectURI)
        setGoogleAccessToken(tokens.accessToken)
        setGoogleRefreshToken(tokens.refreshToken ?? '')
        setOauthConnected(true)
        setFolderStack([])
        setSelectedFileIDs([])
        setSelectedFileMetaByID({})
        await loadFolder('', [])
      } catch (err) {
        setFormError(err instanceof Error ? err.message : 'Failed to finish Google OAuth')
      } finally {
        setOauthConnecting(false)
      }
    }

    window.addEventListener('message', handleMessage)
    return () => {
      window.removeEventListener('message', handleMessage)
    }
  }, [authToken])

  function validate() {
    const e: Record<string, string> = {}
    if (!name.trim()) e.name = 'Required'
    if (providerTab === 'google') {
      if (!oauthConnected || !googleAccessToken) e.google = 'Connect Google Drive first'
      if (importMode === 'selected' && selectedFileIDs.length === 0) {
        e.selectedFileIDs = 'Select at least one file to sync'
      }
    } else {
      if (!rcloneRemote.trim()) e.rcloneRemote = 'Rclone remote is required'
      if (rcloneSelection.length === 0 && !rcloneCurrentPath) {
        e.rcloneSelection = 'Select at least one file/folder or browse to a root path'
      }
    }
    if (syncEnabled && (!autoSyncInterval || Number.parseInt(autoSyncInterval, 10) < 1)) {
      e.autoSyncInterval = 'Must be at least 1 minute'
    }
    return e
  }

  async function handleSave() {
    setFormError('')
    const e = validate()
    if (Object.keys(e).length > 0) {
      setErrors(e)
      return
    }

    const source: DriveSourceDraft = providerTab === 'google'
      ? {
        sourceType: 'google_drive',
        name: name.trim(),
        folderLink: currentFolderID,
        accessToken: googleAccessToken,
        refreshToken: googleRefreshToken,
        clientId: '',
        clientSecret: '',
        selectedFileIDs: importMode === 'selected' ? selectedFileIDs : [],
        selectedFolderIDs: importMode === 'all' && currentFolderID ? [currentFolderID] : [],
        syncEnabled,
        autoSyncInterval: syncEnabled ? Number.parseInt(autoSyncInterval, 10) : 0,
        rcloneRemote: '',
        rcloneRootPath: '',
        rcloneScopePaths: [],
        selectedRclonePaths: [],
        rcloneConfigRef: '',
      }
      : {
        sourceType: 'rclone',
        name: name.trim(),
        folderLink: '',
        accessToken: '',
        refreshToken: '',
        clientId: '',
        clientSecret: '',
        selectedFileIDs: [],
        selectedFolderIDs: [],
        syncEnabled,
        autoSyncInterval: syncEnabled ? Number.parseInt(autoSyncInterval, 10) : 0,
        rcloneRemote: rcloneRemote.trim(),
        rcloneRootPath: rcloneCurrentPath || '',
        rcloneScopePaths: [],
        selectedRclonePaths: Array.from(new Set(rcloneSelection)),
        rcloneConfigRef: rcloneConfigRef.trim(),
      }

    setSaving(true)
    try {
      await onAdd(source)
      setSaved(true)
      setTimeout(() => onClose(), 500)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to add source')
      setSaving(false)
      return
    }
    setSaving(false)
  }

  const selectedFiles = selectedFileIDs.map(id => selectedFileMetaByID[id] ?? {
    id,
    name: id,
    path: id,
    mimeType: '',
  })

  async function openFilePicker() {
    setPickerSelectionIDs(selectedFileIDs)
    setShowFilePicker(true)
    if (!filesLoaded) {
      if (pickerBrowseTab === 'recent') {
        await loadFolder('', [], 'recent', '')
      } else if (pickerBrowseTab === 'shared_drives') {
        await loadFolder(currentFolderID, folderStack, 'shared_drives', pickerSharedDriveID)
      } else {
        await loadFolder(currentFolderID, folderStack, 'folders', '')
      }
    }
  }

  function togglePickerFile(fileID: string, checked: boolean) {
    setPickerSelectionIDs(prev => {
      if (checked) return Array.from(new Set([...prev, fileID]))
      return prev.filter(id => id !== fileID)
    })
  }

  function savePickerSelection() {
    setSelectedFileIDs(Array.from(new Set(pickerSelectionIDs)))
    clearSelectionError()
    setShowFilePicker(false)
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
      void loadFolder('', [], 'folders', '')
      return
    }
  }

  async function uploadPickerFile() {
    if (!pickerUploadFile) {
      setFilesError('Choose a file first.')
      return
    }
    if (!googleAccessToken) {
      setFilesError('Connect Google Drive first.')
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
        accessToken: googleAccessToken,
        refreshToken: googleRefreshToken,
        browseView: 'upload',
        browseFolderID: currentFolderID,
        sharedDriveID: pickerSharedDriveID,
        uploadName: pickerUploadFile.name,
        uploadMimeType: pickerUploadFile.type || 'application/octet-stream',
        uploadContentBase64: btoa(binary),
      })
      setPickerUploadProgress(90)

      setPickerUploadFile(null)
      if (result.files.length > 0) {
        setMetaForFiles(result.files)
        setPickerSelectionIDs(prev => Array.from(new Set([...prev, result.files[0].id])))
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

  function setRcloneSelectionMetaEntry(path: string, name: string, kind: 'file' | 'folder') {
    setRcloneSelectionMetaMap(prev => ({
      ...prev,
      [path]: { path, name, kind },
    }))
  }

  function toggleRcloneSelection(path: string, checked: boolean, name: string, kind: 'file' | 'folder') {
    setRcloneSelection(prev => {
      if (checked) return Array.from(new Set([...prev, path]))
      return prev.filter(item => item !== path)
    })
    setRcloneSelectionMetaEntry(path, name, kind)
    clearFieldError('rcloneSelection')
  }

  async function loadRclonePath(path?: string) {
    if (!rcloneRemote.trim()) {
      setRcloneError('Enter rclone remote first (example: gdrive).')
      return
    }
    setRcloneLoading(true)
    setRcloneError('')
    try {
      const result = await browseRclonePath(authToken, rcloneRemote.trim(), path)
      setRcloneCurrentPath(result.currentPath ?? '')
      setRcloneParentPath(result.parentPath)
      setRcloneFolders(result.folders)
      setRcloneFiles(result.files)
      setRcloneLoaded(true)
    } catch (err) {
      setRcloneError(err instanceof Error ? err.message : 'Failed to browse rclone path')
      setRcloneFolders([])
      setRcloneFiles([])
      setRcloneLoaded(false)
    } finally {
      setRcloneLoading(false)
    }
  }

  function rclonePathParts() {
    const current = rcloneCurrentPath.replace(/^\/+|\/+$/g, '')
    if (!current) return [{ label: '/', path: '' }]
    const parts = current.split('/')
    const out: Array<{ label: string; path: string }> = [{ label: '/', path: '' }]
    let acc = ''
    for (const part of parts) {
      acc = acc ? `${acc}/${part}` : part
      out.push({ label: part, path: acc })
    }
    return out
  }

  const rcloneBreadcrumbs = rclonePathParts()

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(7,12,22,0.85)',
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'center',
        zIndex: 100,
        backdropFilter: 'blur(4px)',
        overflowY: 'auto',
        padding: '20px 12px',
        boxSizing: 'border-box',
      }}
      onClick={onClose}
    >
      <div
        style={{
          position: 'relative',
          background: 'var(--card-bg)',
          border: '1px solid var(--border)',
          borderRadius: '16px',
          width: '100%',
          maxWidth: '720px',
          maxHeight: 'calc(100vh - 40px)',
          display: 'flex',
          flexDirection: 'column',
          boxShadow: '0 24px 80px rgba(0,0,0,0.6)',
          margin: 'auto 0',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div style={{ padding: '22px 24px 18px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', gap: '10px', flexShrink: 0 }}>
          <div style={{ width: '32px', height: '32px', borderRadius: '8px', background: 'rgba(66,133,244,0.12)', border: '1px solid rgba(66,133,244,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <GoogleDriveIcon size={18} />
          </div>
          <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '16px', color: 'var(--text)', flex: 1 }}>
            Add Source
          </span>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px', display: 'flex', borderRadius: '6px' }}>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/></svg>
          </button>
        </div>

        <div style={{ padding: '22px 24px', display: 'flex', flexDirection: 'column', gap: '16px', overflowY: 'auto', flex: 1 }}>
          <Field label="Source Name *" error={errors.name}>
            <input
              type="text"
              placeholder="e.g. Product Docs"
              value={name}
              onChange={e => {
                setName(e.target.value)
                clearFieldError('name')
              }}
              style={{ ...inputStyle, borderColor: errors.name ? 'rgba(255,107,107,0.5)' : undefined }}
              autoFocus
            />
          </Field>

          <div style={{ display: 'flex', gap: '8px', padding: '4px', border: '1px solid var(--border)', borderRadius: '10px', background: 'rgba(255,255,255,0.02)' }}>
            <button
              type="button"
              onClick={() => setProviderTab('google')}
              style={{
                flex: 1,
                borderRadius: '8px',
                border: '1px solid var(--border)',
                background: providerTab === 'google' ? 'rgba(0,212,180,0.12)' : 'transparent',
                color: providerTab === 'google' ? 'var(--accent)' : 'var(--text-muted)',
                cursor: 'pointer',
                fontFamily: 'Space Grotesk, sans-serif',
                fontSize: '12px',
                fontWeight: 600,
                padding: '8px 10px',
              }}
            >
              Google Drive (Recommended)
            </button>
            <button
              type="button"
              onClick={() => setProviderTab('rclone')}
              style={{
                flex: 1,
                borderRadius: '8px',
                border: '1px solid var(--border)',
                background: providerTab === 'rclone' ? 'rgba(0,212,180,0.12)' : 'transparent',
                color: providerTab === 'rclone' ? 'var(--accent)' : 'var(--text-muted)',
                cursor: 'pointer',
                fontFamily: 'Space Grotesk, sans-serif',
                fontSize: '12px',
                fontWeight: 600,
                padding: '8px 10px',
              }}
            >
              Other Clouds (rclone)
            </button>
          </div>

          {providerTab === 'google' && (
            <>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', padding: '12px 14px', background: 'rgba(255,255,255,0.025)', borderRadius: '10px', border: `1px solid ${errors.google ? 'rgba(255,107,107,0.5)' : 'var(--border)'}` }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '10px' }}>
              <div>
                <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>
                  Google Drive Connection
                </div>
                <div style={{ ...hintStyle, marginTop: '2px' }}>
                  Connect once, then browse folders and select files.
                </div>
              </div>
              <button
                type="button"
                onClick={() => { void connectGoogleDrive() }}
                disabled={oauthConnecting}
                style={{ background: oauthConnected ? 'rgba(0,212,180,0.12)' : 'none', border: `1px solid ${oauthConnected ? 'rgba(0,212,180,0.35)' : 'var(--border)'}`, color: oauthConnected ? '#00d4b4' : 'var(--text)', padding: '7px 10px', borderRadius: '8px', cursor: oauthConnecting ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', opacity: oauthConnecting ? 0.8 : 1 }}
              >
                {oauthConnecting ? 'Connecting…' : oauthConnected ? 'Connected' : 'Connect Google'}
              </button>
            </div>
            {errors.google && <div style={errorStyle}>{errors.google}</div>}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', padding: '12px 14px', background: 'rgba(255,255,255,0.03)', borderRadius: '10px', border: `1px solid ${errors.selectedFileIDs ? 'rgba(255,107,107,0.5)' : 'var(--border)'}` }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '10px' }}>
              <div>
                <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>
                  File Selection
                </div>
                <div style={{ ...hintStyle, marginTop: '2px' }}>
                  {filesLoaded ? `Current folder: ${folderPathLabel}` : 'Open picker and select files.'}
                </div>
              </div>
              <button
                type="button"
                onClick={() => { void openFilePicker() }}
                disabled={filesLoading || !oauthConnected}
                style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text)', padding: '7px 10px', borderRadius: '8px', cursor: filesLoading || !oauthConnected ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', opacity: filesLoading || !oauthConnected ? 0.7 : 1 }}
              >
                {filesLoading ? 'Loading…' : filesLoaded ? 'Open Picker' : 'Load'}
              </button>
            </div>

            <div style={{ minHeight: selectedFiles.length > 0 ? '104px' : '84px', borderRadius: '8px', border: '1px dashed var(--border)', background: 'rgba(255,255,255,0.02)', padding: '10px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {selectedFiles.length === 0 ? (
                <div style={{ ...hintStyle, marginTop: 0 }}>
                  No files selected yet.
                </div>
              ) : (
                selectedFiles.map(file => (
                  <div key={file.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, border: '1px solid var(--border)', borderRadius: '7px', padding: '6px 8px', background: 'rgba(255,255,255,0.03)' }}>
                    <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)', flexShrink: 0 }}>FILE</span>
                    <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', flex: 1 }}>
                      {file.name}
                    </span>
                    <button
                      type="button"
                      onClick={() => removeSelectedFile(file.id)}
                      style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: '2px 4px', borderRadius: '5px', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
                    >
                      Remove
                    </button>
                  </div>
                ))
              )}
            </div>

            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
              <div style={hintStyle}>{selectedFileIDs.length} files selected</div>
              <button
                type="button"
                onClick={() => {
                  setSelectedFileIDs([])
                  setSelectedFileMetaByID({})
                  clearSelectionError()
                }}
                style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', padding: '5px 8px', borderRadius: '7px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
              >
                Clear selection
              </button>
            </div>

            {filesError && <div style={errorStyle}>{filesError}</div>}
            {errors.selectedFileIDs && <div style={errorStyle}>{errors.selectedFileIDs}</div>}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', padding: '12px 14px', background: 'rgba(255,255,255,0.025)', borderRadius: '10px', border: '1px solid var(--border)' }}>
            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>
              Import Mode
            </div>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
              <input
                type="radio"
                name="google-import-mode"
                checked={importMode === 'selected'}
                onChange={() => setImportMode('selected')}
                style={{ accentColor: 'var(--accent)' }}
              />
              <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text)' }}>
                Choose files manually
              </span>
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
              <input
                type="radio"
                name="google-import-mode"
                checked={importMode === 'all'}
                onChange={() => setImportMode('all')}
                style={{ accentColor: 'var(--accent)' }}
              />
              <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '11px', color: 'var(--text)' }}>
                Import all files in current folder (recursive)
              </span>
            </label>
          </div>
            </>
          )}

          {providerTab === 'rclone' && (
            <>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', padding: '12px 14px', background: 'rgba(255,255,255,0.025)', borderRadius: '10px', border: `1px solid ${errors.rcloneRemote ? 'rgba(255,107,107,0.5)' : 'var(--border)'}` }}>
                <Field label="Rclone Remote *" hint="Example: gdrive, onedrive, dropbox, s3" error={errors.rcloneRemote}>
                  <input
                    type="text"
                    placeholder="gdrive"
                    value={rcloneRemote}
                    onChange={e => {
                      setRcloneRemote(e.target.value)
                      clearFieldError('rcloneRemote')
                    }}
                    style={{ ...inputStyle, borderColor: errors.rcloneRemote ? 'rgba(255,107,107,0.5)' : undefined }}
                  />
                </Field>

                <Field label="Config Ref (optional)" hint="Optional config reference if backend expects it.">
                  <input
                    type="text"
                    placeholder="secret/rclone"
                    value={rcloneConfigRef}
                    onChange={e => setRcloneConfigRef(e.target.value)}
                    style={inputStyle}
                  />
                </Field>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', padding: '12px 14px', background: 'rgba(255,255,255,0.03)', borderRadius: '10px', border: `1px solid ${errors.rcloneSelection ? 'rgba(255,107,107,0.5)' : 'var(--border)'}` }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '10px' }}>
                  <div>
                    <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>
                      Rclone Browser
                    </div>
                    <div style={{ ...hintStyle, marginTop: '2px' }}>
                      Browse remote and select files/folders to sync.
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => { void loadRclonePath(rcloneCurrentPath || undefined) }}
                    disabled={rcloneLoading || !rcloneRemote.trim()}
                    style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text)', padding: '7px 10px', borderRadius: '8px', cursor: rcloneLoading || !rcloneRemote.trim() ? 'default' : 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', opacity: rcloneLoading || !rcloneRemote.trim() ? 0.7 : 1 }}
                  >
                    {rcloneLoading ? 'Loading…' : rcloneLoaded ? 'Refresh' : 'Load'}
                  </button>
                </div>

                {rcloneLoaded && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    {rcloneBreadcrumbs.map((crumb, index) => (
                      <div key={`${crumb.path || 'root'}-${index}`} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                        {index > 0 && <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>/</span>}
                        <button
                          type="button"
                          onClick={() => { void loadRclonePath(crumb.path || undefined) }}
                          style={{ background: index === rcloneBreadcrumbs.length - 1 ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.04)', border: '1px solid var(--border)', color: 'var(--text)', padding: '4px 7px', borderRadius: '7px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
                        >
                          {crumb.label}
                        </button>
                      </div>
                    ))}
                    {rcloneParentPath !== undefined && (
                      <button
                        type="button"
                        onClick={() => { void loadRclonePath(rcloneParentPath || undefined) }}
                        style={{ marginLeft: 'auto', background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', padding: '4px 7px', borderRadius: '7px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
                      >
                        Up
                      </button>
                    )}
                  </div>
                )}

                {rcloneLoaded && (
                  <input
                    type="text"
                    placeholder="Filter folders/files..."
                    value={rcloneSearch}
                    onChange={e => setRcloneSearch(e.target.value)}
                    style={{ ...inputStyle, fontSize: '12px' }}
                  />
                )}

                {rcloneLoaded && (
                  <div style={{ maxHeight: '260px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: '8px' }}>
                    {visibleRcloneFolders.map(folder => {
                      const checked = rcloneSelection.includes(folder.path)
                      return (
                        <label key={`dir-${folder.path}`} style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '8px 10px', borderBottom: '1px solid var(--border)', cursor: 'pointer' }}>
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={e => toggleRcloneSelection(folder.path, e.target.checked, folder.name, 'folder')}
                            style={{ marginTop: '2px', accentColor: 'var(--accent)' }}
                          />
                          <div style={{ minWidth: 0 }}>
                            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>[DIR]</div>
                            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '520px' }}>
                              {folder.name}
                            </div>
                            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{folder.path || '/'}</div>
                          </div>
                        </label>
                      )
                    })}

                    {visibleRcloneFiles.map(file => {
                      const checked = rcloneSelection.includes(file.path)
                      return (
                        <label key={`file-${file.path}`} style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', padding: '8px 10px', borderBottom: '1px solid var(--border)', cursor: 'pointer' }}>
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={e => toggleRcloneSelection(file.path, e.target.checked, file.name, 'file')}
                            style={{ marginTop: '2px', accentColor: 'var(--accent)' }}
                          />
                          <div style={{ minWidth: 0 }}>
                            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>[FILE]</div>
                            <div style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: '520px' }}>
                              {file.name}
                            </div>
                            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{file.path}</div>
                          </div>
                        </label>
                      )
                    })}

                    {visibleRcloneFolders.length === 0 && visibleRcloneFiles.length === 0 && (
                      <div style={{ ...hintStyle, padding: '10px' }}>No items in this path.</div>
                    )}
                  </div>
                )}

                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
                  <div style={hintStyle}>{rcloneSelection.length} paths selected</div>
                  <button
                    type="button"
                    onClick={() => {
                      setRcloneSelection([])
                      setRcloneSelectionMetaMap({})
                      clearFieldError('rcloneSelection')
                    }}
                    style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', padding: '5px 8px', borderRadius: '7px', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
                  >
                    Clear selection
                  </button>
                </div>

                {rcloneSelection.length > 0 && (
                  <div style={{ maxHeight: '100px', overflowY: 'auto', border: '1px solid var(--border)', borderRadius: '8px', padding: '6px 8px', background: 'rgba(255,255,255,0.02)' }}>
                    {rcloneSelection.map(path => {
                      const meta = rcloneSelectionMeta[path]
                      return (
                        <div key={path} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 0', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                          <span style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: 'var(--text-dim)' }}>{meta?.kind === 'folder' ? '[DIR]' : '[FILE]'}</span>
                          <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', flex: 1 }}>{meta?.name ?? path}</span>
                          <button
                            type="button"
                            onClick={() => toggleRcloneSelection(path, false, meta?.name ?? path, meta?.kind ?? 'file')}
                            style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px' }}
                          >
                            Remove
                          </button>
                        </div>
                      )
                    })}
                  </div>
                )}

                {rcloneError && <div style={errorStyle}>{rcloneError}</div>}
                {errors.rcloneSelection && <div style={errorStyle}>{errors.rcloneSelection}</div>}
              </div>
            </>
          )}

          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', padding: '14px 16px', background: 'rgba(255,255,255,0.025)', borderRadius: '10px', border: '1px solid var(--border)' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={syncEnabled}
                onChange={e => setSyncEnabled(e.target.checked)}
                style={{ width: '15px', height: '15px', accentColor: 'var(--accent)', cursor: 'pointer' }}
              />
              <span style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '600', fontSize: '13px', color: 'var(--text)' }}>
                Enable auto-sync
              </span>
            </label>
            {syncEnabled && (
              <Field label="Sync interval (minutes)" error={errors.autoSyncInterval}>
                <input
                  type="number"
                  min="1"
                  placeholder="60"
                  value={autoSyncInterval}
                  onChange={e => setAutoSyncInterval(e.target.value)}
                  style={{ ...inputStyle, width: '140px', borderColor: errors.autoSyncInterval ? 'rgba(255,107,107,0.5)' : undefined }}
                />
              </Field>
            )}
          </div>

          {formError && (
            <div style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', color: '#ff6b6b' }}>
              {formError}
            </div>
          )}
        </div>

        <div style={{ padding: '16px 24px', borderTop: '1px solid var(--border)', display: 'flex', justifyContent: 'flex-end', gap: '10px', flexShrink: 0 }}>
          <button
            onClick={onClose}
            style={{ background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', padding: '8px 18px', borderRadius: '8px', cursor: 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '500' }}
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving || saved}
            style={{ background: saved ? 'rgba(0,212,180,0.7)' : 'var(--accent)', color: '#070c16', padding: '8px 20px', border: 'none', borderRadius: '8px', cursor: saving || saved ? 'default' : 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', fontWeight: '700', opacity: saving ? 0.8 : 1, minWidth: '80px' }}
          >
            {saved ? '✓ Saved' : saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>

      {showFilePicker && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.35)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 101,
            padding: '16px',
            boxSizing: 'border-box',
          }}
          onClick={() => setShowFilePicker(false)}
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
              <button
                type="button"
                onClick={() => setShowFilePicker(false)}
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
                  onClick={() => setPickerViewMode('cards')}
                  style={{ height: '30px', minWidth: '56px', border: 'none', borderRight: '1px solid var(--border)', background: pickerViewMode === 'cards' ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.03)', color: pickerViewMode === 'cards' ? 'var(--text)' : 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                >
                  Cards
                </button>
                <button
                  type="button"
                  onClick={() => setPickerViewMode('list')}
                  style={{ height: '30px', minWidth: '56px', border: 'none', background: pickerViewMode === 'list' ? 'rgba(0,212,180,0.12)' : 'rgba(255,255,255,0.03)', color: pickerViewMode === 'list' ? 'var(--text)' : 'var(--text-dim)', fontFamily: 'JetBrains Mono, monospace', fontSize: '10px', cursor: 'pointer' }}
                >
                  List
                </button>
              </div>
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
                    Upload file to {folderPathLabel}
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

              {pickerViewMode === 'cards' ? (
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(185px, 1fr))', gap: '12px' }}>
                  {shownFiles.map(file => {
                    const selected = pickerSelectionIDs.includes(file.id)
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
                            togglePickerFile(file.id, e.target.checked)
                            setMetaForFiles([file])
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
                    const selected = pickerSelectionIDs.includes(file.id)
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
                            togglePickerFile(file.id, e.target.checked)
                            setMetaForFiles([file])
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
                {pickerSelectionIDs.length} files marked
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <button
                  type="button"
                  onClick={savePickerSelection}
                  style={{ minWidth: '82px', height: '30px', border: 'none', borderRadius: '6px', background: 'var(--accent)', color: '#070c16', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 700, cursor: 'pointer' }}
                >
                  Select
                </button>
                <button
                  type="button"
                  onClick={() => setShowFilePicker(false)}
                  style={{ minWidth: '82px', height: '30px', border: '1px solid var(--border)', borderRadius: '6px', background: 'rgba(255,255,255,0.03)', color: 'var(--text-muted)', fontFamily: 'Space Grotesk, sans-serif', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}
                >
                  Cancel
                </button>
              </div>
            </div>

            {filesError && (
              <div style={{ ...errorStyle, padding: '10px 16px', borderTop: '1px solid var(--border)', marginTop: 0 }}>
                {filesError}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
