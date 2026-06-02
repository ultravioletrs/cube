// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AppRecord, ChatMessage, Conversation, DriveSource, DriveSourceDraft, RecordFormat } from '@/types'

interface RecordDTO {
  id: string
  source_id?: string
  name: string
  format: string
  status: string
  created_at: string
  error?: string | null
  description?: string
  chunks?: number | null
  ingest_total_chunks?: number | null
  ingest_indexed_chunks?: number | null
  size_bytes?: number | null
  pages?: number | null
  external_url?: string
}

interface SourceDTO {
  id: string
  source_type: string
  name: string
  config?: Record<string, unknown>
  status: string
  sync_enabled: boolean
  auto_sync_interval: number
  last_sync_error?: string
  last_sync_at?: string
  created_at: string
}

interface RecordListDTO {
  records: RecordDTO[]
}

interface SourceListDTO {
  sources: SourceDTO[]
}

interface SourceSyncDTO {
  source: SourceDTO
  discovered: number
  queued: number
  updated: number
  unchanged: number
  deleted: number
}

interface RcloneBrowseFileDTO {
  name: string
  path: string
  mime_type: string
  size: number
  modified_time?: string
}

interface RcloneBrowseFolderDTO {
  name: string
  path: string
}

interface RcloneBrowseDTO {
  current_path: string
  parent_path?: string
  folders: RcloneBrowseFolderDTO[]
  files: RcloneBrowseFileDTO[]
}

interface GoogleOAuthURLDTO {
  auth_url: string
  state: string
}

interface GoogleOAuthExchangeDTO {
  access_token: string
  refresh_token?: string
  expires_in?: number
  token_type?: string
  scope?: string
}

interface GoogleDriveFileDTO {
  id: string
  name: string
  mime_type: string
  modified_time: string
  web_view_link: string
  icon_link?: string
  thumbnail_link?: string
  parents: string[]
}

interface GoogleDriveBrowseDTO {
  files: GoogleDriveFileDTO[]
  folders: GoogleDriveFileDTO[]
  current_folder_id?: string
  current_shared_drive_id?: string
  requested_scope?: string
}

interface APIErrorDTO {
  error?: string
  message?: string
}

interface ConversationDTO {
  id: string
  title: string
  created_at: string
  updated_at: string
}

interface ConversationListDTO {
  conversations: ConversationDTO[]
}

interface ConversationMessageDTO {
  id: string
  conversation_id: string
  role: string
  content: string
  created_at: string
}

interface ConversationDetailDTO {
  conversation: ConversationDTO
  messages: ConversationMessageDTO[]
}

interface ChatMatchDTO {
  chunk_id: string
  record_id: string
  record_name: string
  record_format: string
  source_id?: string
  source_name?: string
  chunk_index: number
  page_number?: number
  content: string
}

interface ChatResponseDTO {
  query: string
  mode: string
  matches: ChatMatchDTO[]
  total: number
}

export interface ChatMatch {
  chunkID: string
  recordID: string
  recordName: string
  recordFormat: RecordFormat
  sourceID?: string
  sourceName?: string
  chunkIndex: number
  pageNumber?: number
  content: string
}

export interface DriveFileOption {
  id: string
  name: string
  path: string
  mimeType: string
  modifiedTime?: string
  size?: number
  iconLink?: string
  thumbnailLink?: string
}

export interface DriveFolderOption {
  id: string
  name: string
  mimeType?: string
  modifiedTime: string
  webViewLink: string
  parents: string[]
}

export interface DriveBrowseResult { files: DriveFileOption[] }

export interface RcloneFolderOption {
  name: string
  path: string
}

export interface RcloneBrowseResult {
  currentPath: string
  parentPath?: string
  folders: RcloneFolderOption[]
  files: DriveFileOption[]
}

export interface GoogleOAuthTokens {
  accessToken: string
  refreshToken?: string
}

export interface GoogleDriveBrowseResult {
  files: DriveFileOption[]
  folders: DriveFolderOption[]
  currentFolderID?: string
  currentSharedDriveID?: string
  requestedScope?: string
}

export interface RecordListOptions {
  status?: AppRecord['status'] | 'all'
  format?: RecordFormat | 'all'
  sourceID?: string
  limit?: number
  offset?: number
}

type RawRecordStatus = 'queued' | 'processing' | 'indexed' | 'failed' | 'cancelled'

const embedderURL = import.meta.env.VITE_EMBEDDER_URL ?? window.location.origin

function buildURL(path: string): string {
  return new URL(path, embedderURL).toString()
}

function formatDate(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function bytesToLabel(value?: number | null): string | undefined {
  if (value == null || value <= 0) return undefined
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unitIdx = 0
  while (size >= 1024 && unitIdx < units.length - 1) {
    size /= 1024
    unitIdx++
  }
  const fixed = unitIdx === 0 ? size.toFixed(0) : size.toFixed(1)
  return `${fixed} ${units[unitIdx]}`
}

function toRecordStatus(status: string): AppRecord['status'] {
  switch (status) {
    case 'indexed':
      return 'indexed'
    case 'failed':
      return 'error'
    case 'cancelled':
      return 'cancelled'
    case 'queued':
    case 'processing':
    default:
      return 'processing'
  }
}

function toRecordFormat(format: string): RecordFormat {
  switch (format) {
    case 'text':
    case 'pdf':
    case 'md':
    case 'docx':
    case 'code':
    case 'image':
      return format
    default:
      return 'link'
  }
}

function mapRecord(dto: RecordDTO): AppRecord {
  return {
    id: dto.id,
    sourceID: dto.source_id,
    name: dto.name,
    format: toRecordFormat(dto.format),
    status: toRecordStatus(dto.status),
    createdAt: formatDate(dto.created_at),
    error: dto.error ?? undefined,
    description: dto.description ?? '',
    chunks: dto.chunks ?? null,
    ingestTotalChunks: dto.ingest_total_chunks ?? null,
    ingestIndexedChunks: dto.ingest_indexed_chunks ?? null,
    pages: dto.pages ?? null,
    size: bytesToLabel(dto.size_bytes ?? undefined),
    url: dto.external_url || undefined,
  }
}

function mapSource(dto: SourceDTO): DriveSource {
  const config = dto.config ?? {}
  const sourceType = dto.source_type === 'google_drive' ? 'google_drive' : 'rclone'
  const folderLink = typeof config.folder_link === 'string'
    ? config.folder_link
    : typeof config.folder_id === 'string'
      ? config.folder_id
      : ''
  const selectedFileIDs = Array.isArray(config.selected_file_ids)
    ? config.selected_file_ids.filter((v): v is string => typeof v === 'string')
    : []
  const selectedFolderIDs = Array.isArray(config.selected_folder_ids)
    ? config.selected_folder_ids.filter((v): v is string => typeof v === 'string')
    : []
  const accessToken = typeof config.access_token === 'string' ? config.access_token : ''
  const refreshToken = typeof config.refresh_token === 'string' ? config.refresh_token : ''
  const clientId = typeof config.client_id === 'string' ? config.client_id : ''
  const clientSecret = typeof config.client_secret === 'string' ? config.client_secret : ''
  const rcloneRemote = typeof config.remote === 'string' ? config.remote : ''
  const rcloneRootPath = typeof config.root_path === 'string' ? config.root_path : ''
  const rcloneScopePaths = Array.isArray(config.scope_paths)
    ? config.scope_paths.filter((v): v is string => typeof v === 'string')
    : []
  const selectedRclonePaths = Array.isArray(config.selected_paths)
    ? config.selected_paths.filter((v): v is string => typeof v === 'string')
    : []
  const rcloneConfigRef = typeof config.config_ref === 'string' ? config.config_ref : ''

  return {
    id: dto.id,
    sourceType,
    name: dto.name,
    rcloneRemote,
    rcloneRootPath,
    rcloneScopePaths,
    selectedRclonePaths,
    rcloneConfigRef,
    folderLink,
    accessToken,
    refreshToken,
    clientId,
    clientSecret,
    selectedFileIDs,
    selectedFolderIDs,
    syncEnabled: dto.sync_enabled,
    autoSyncInterval: dto.auto_sync_interval,
    status: (dto.status === 'syncing' || dto.status === 'error' || dto.status === 'disconnected') ? dto.status : 'active',
    lastSyncError: dto.last_sync_error,
    lastSyncAt: dto.last_sync_at ? formatDate(dto.last_sync_at) : undefined,
    createdAt: formatDate(dto.created_at),
  }
}

async function readError(res: Response): Promise<string> {
  try {
    const body = await res.json() as APIErrorDTO
    return body.error || body.message || `request failed (${res.status})`
  } catch {
    return `request failed (${res.status})`
  }
}

function domainInit(domainID: string, init?: RequestInit): RequestInit {
  return {
    ...init,
    headers: {
      ...(init?.headers ?? {}),
      ...(domainID ? { 'X-Domain-ID': domainID } : {}),
    },
  }
}

function authHeaders(token: string, domainID: string): Record<string, string> {
  const h: Record<string, string> = { Authorization: `Bearer ${token}` }
  if (domainID) h['X-Domain-ID'] = domainID
  return h
}

async function apiJSON<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const res = await fetch(buildURL(path), {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
  return await res.json() as T
}

function recordListPath(opts: RecordListOptions, status?: RawRecordStatus): string {
  const params = new URLSearchParams()
  params.set('limit', String(opts.limit ?? 1000))
  if (opts.offset) params.set('offset', String(opts.offset))
  if (status) params.set('status', status)
  if (opts.format && opts.format !== 'all') params.set('format', opts.format)
  if (opts.sourceID) params.set('source_id', opts.sourceID)

  return `/api/v1/records?${params.toString()}`
}

async function fetchRecords(token: string, domainID: string, opts: RecordListOptions, status?: RawRecordStatus): Promise<RecordDTO[]> {
  const data = await apiJSON<RecordListDTO>(recordListPath(opts, status), token, domainInit(domainID, { method: 'GET' }))
  return data.records
}

export async function listRecords(token: string, domainID: string, opts: RecordListOptions = {}): Promise<AppRecord[]> {
  if (opts.status === 'processing') {
    const records = await Promise.all([
      fetchRecords(token, domainID, opts, 'queued'),
      fetchRecords(token, domainID, opts, 'processing'),
    ])
    return records
      .flat()
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .map(mapRecord)
  }

  const status = opts.status === 'error'
    ? 'failed'
    : opts.status && opts.status !== 'all'
      ? opts.status
      : undefined
  const records = await fetchRecords(token, domainID, opts, status)
  return records.map(mapRecord)
}

export async function listRecordsBySource(token: string, domainID: string, sourceID: string): Promise<AppRecord[]> {
  const data = await apiJSON<RecordListDTO>(`/api/v1/sources/${sourceID}/records?limit=1000`, token, domainInit(domainID, { method: 'GET' }))
  return data.records.map(mapRecord)
}

export async function listSources(token: string, domainID: string): Promise<DriveSource[]> {
  const data = await apiJSON<SourceListDTO>('/api/v1/sources', token, domainInit(domainID, { method: 'GET' }))
  return data.sources.map(mapSource)
}

export async function createSource(token: string, domainID: string, source: DriveSourceDraft): Promise<DriveSource> {
  const config = source.sourceType === 'google_drive'
    ? {
      folder_id: source.folderLink ?? '',
      selected_file_ids: source.selectedFileIDs ?? [],
      selected_folder_ids: source.selectedFolderIDs ?? [],
      access_token: source.accessToken ?? '',
      refresh_token: source.refreshToken ?? '',
      client_id: source.clientId ?? '',
      client_secret: source.clientSecret ?? '',
    }
    : {
      remote: source.rcloneRemote ?? '',
      root_path: source.rcloneRootPath ?? '',
      scope_paths: source.rcloneScopePaths ?? [],
      selected_paths: source.selectedRclonePaths ?? [],
      config_ref: source.rcloneConfigRef ?? '',
    }

  const created = await apiJSON<SourceDTO>('/api/v1/sources', token, domainInit(domainID, {
    method: 'POST',
    body: JSON.stringify({
      source_type: source.sourceType,
      name: source.name,
      sync_enabled: source.syncEnabled,
      auto_sync_interval: source.autoSyncInterval,
      config,
    }),
  }))

  return mapSource(created)
}

export async function getGoogleOAuthURL(token: string, redirectURI: string): Promise<{ authURL: string; state: string }> {
  const data = await apiJSON<GoogleOAuthURLDTO>('/api/v1/sources/google/oauth/url', token, {
    method: 'POST',
    body: JSON.stringify({ redirect_uri: redirectURI }),
  })
  return {
    authURL: data.auth_url,
    state: data.state,
  }
}

export async function exchangeGoogleOAuth(
  token: string,
  code: string,
  state: string,
  redirectURI: string,
): Promise<GoogleOAuthTokens> {
  const data = await apiJSON<GoogleOAuthExchangeDTO>('/api/v1/sources/google/oauth/exchange', token, {
    method: 'POST',
    body: JSON.stringify({
      code,
      state,
      redirect_uri: redirectURI,
    }),
  })

  return {
    accessToken: data.access_token,
    refreshToken: data.refresh_token,
  }
}

export async function browseGoogleDrive(
  token: string,
  opts: {
    accessToken: string
    refreshToken?: string
    clientID?: string
    clientSecret?: string
    browseFolderID?: string
    browseView?: 'folders' | 'recent' | 'shared_drives' | 'upload'
    sharedDriveID?: string
    uploadName?: string
    uploadMimeType?: string
    uploadContentBase64?: string
    folderID?: string
    folderLink?: string
  },
): Promise<GoogleDriveBrowseResult> {
  const data = await apiJSON<GoogleDriveBrowseDTO>('/api/v1/sources/google/files', token, {
    method: 'POST',
    body: JSON.stringify({
      access_token: opts.accessToken,
      refresh_token: opts.refreshToken ?? '',
      client_id: opts.clientID ?? '',
      client_secret: opts.clientSecret ?? '',
      browse_mode: true,
      browse_folder_id: opts.browseFolderID ?? '',
      browse_view: opts.browseView ?? 'folders',
      shared_drive_id: opts.sharedDriveID ?? '',
      upload_name: opts.uploadName ?? '',
      upload_mime_type: opts.uploadMimeType ?? '',
      upload_content_base64: opts.uploadContentBase64 ?? '',
      folder_id: opts.folderID ?? '',
      folder_link: opts.folderLink ?? '',
    }),
  })

  return {
    currentFolderID: data.current_folder_id,
    currentSharedDriveID: data.current_shared_drive_id,
    requestedScope: data.requested_scope,
    folders: (data.folders ?? []).map(folder => ({
      id: folder.id,
      name: folder.name,
      mimeType: folder.mime_type,
      modifiedTime: folder.modified_time,
      webViewLink: folder.web_view_link,
      parents: folder.parents,
    })),
    files: (data.files ?? []).map(file => ({
      id: file.id,
      name: file.name,
      path: file.name,
      mimeType: file.mime_type,
      modifiedTime: file.modified_time,
      iconLink: file.icon_link,
      thumbnailLink: file.thumbnail_link,
    })),
  }
}

export async function listRcloneFilesForSource(
  token: string,
  source: Pick<DriveSourceDraft, 'rcloneRemote' | 'rcloneRootPath' | 'rcloneScopePaths'>,
): Promise<DriveBrowseResult> {
  const data = await apiJSON<RcloneBrowseDTO>('/api/v1/sources/rclone/files', token, {
    method: 'POST',
    body: JSON.stringify({
      remote: source.rcloneRemote ?? '',
      root_path: source.rcloneRootPath ?? '',
      scope_paths: source.rcloneScopePaths ?? [],
    }),
  })

  return {
    files: data.files.map(file => ({
      id: file.path,
      name: file.name,
      path: file.path,
      mimeType: file.mime_type,
      modifiedTime: file.modified_time,
      size: file.size,
    })),
  }
}

export async function browseRclonePath(
  token: string,
  remote: string,
  path?: string,
): Promise<RcloneBrowseResult> {
  const data = await apiJSON<RcloneBrowseDTO>('/api/v1/sources/rclone/browse', token, {
    method: 'POST',
    body: JSON.stringify({
      remote,
      path: path ?? '',
    }),
  })

  return {
    currentPath: data.current_path,
    parentPath: data.parent_path,
    folders: data.folders.map(folder => ({
      name: folder.name,
      path: folder.path,
    })),
    files: data.files.map(file => ({
      id: file.path,
      name: file.name,
      path: file.path,
      mimeType: file.mime_type,
      modifiedTime: file.modified_time,
      size: file.size,
    })),
  }
}

export interface SourceSyncResult {
  source: DriveSource
  discovered: number
  queued: number
  updated: number
  unchanged: number
  deleted: number
}

export async function syncSource(token: string, domainID: string, sourceID: string): Promise<SourceSyncResult> {
  const data = await apiJSON<SourceSyncDTO>(`/api/v1/sources/${sourceID}/sync`, token, domainInit(domainID, {
    method: 'POST',
  }))
  return {
    source: mapSource(data.source),
    discovered: data.discovered,
    queued: data.queued,
    updated: data.updated,
    unchanged: data.unchanged,
    deleted: data.deleted ?? 0,
  }
}

export async function updateGoogleSourceSelection(
  token: string,
  domainID: string,
  sourceID: string,
  selectedFileIDs: string[],
  selectedFolderIDs: string[] = [],
): Promise<DriveSource> {
  const updated = await apiJSON<SourceDTO>(`/api/v1/sources/${sourceID}/selection`, token, domainInit(domainID, {
    method: 'PUT',
    body: JSON.stringify({
      selected_file_ids: selectedFileIDs,
      selected_folder_ids: selectedFolderIDs,
    }),
  }))
  return mapSource(updated)
}

export async function deleteSource(token: string, domainID: string, sourceID: string): Promise<void> {
  const res = await fetch(buildURL(`/api/v1/sources/${sourceID}`), {
    method: 'DELETE',
    headers: authHeaders(token, domainID),
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
}

export async function deleteRecord(token: string, domainID: string, recordID: string): Promise<void> {
  const res = await fetch(buildURL(`/api/v1/records/${recordID}`), {
    method: 'DELETE',
    headers: authHeaders(token, domainID),
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
}

export async function retryRecordIngest(token: string, domainID: string, recordID: string): Promise<void> {
  const res = await fetch(buildURL(`/api/v1/records/${recordID}/retry`), {
    method: 'POST',
    headers: authHeaders(token, domainID),
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
}

export async function cancelRecordIngest(token: string, domainID: string, recordID: string): Promise<void> {
  const res = await fetch(buildURL(`/api/v1/records/${recordID}/cancel`), {
    method: 'POST',
    headers: authHeaders(token, domainID),
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
}

export async function uploadRecordFile(
  token: string,
  domainID: string,
  file: File,
  name?: string,
  sourceID?: string,
): Promise<AppRecord> {
  const formData = new FormData()
  formData.append('file', file)
  if (name?.trim()) formData.append('name', name.trim())
  if (sourceID?.trim()) formData.append('source_id', sourceID.trim())

  const res = await fetch(buildURL('/api/v1/records/upload'), {
    method: 'POST',
    headers: authHeaders(token, domainID),
    body: formData,
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }

  const dto = await res.json() as RecordDTO
  return mapRecord(dto)
}

function mapConversation(dto: ConversationDTO): Conversation {
  return {
    id: dto.id,
    title: dto.title || 'Untitled',
    createdAt: formatDate(dto.created_at),
    updatedAt: formatDate(dto.updated_at),
  }
}

export async function listConversations(token: string, domainID: string): Promise<Conversation[]> {
  const data = await apiJSON<ConversationListDTO>('/api/v1/conversations', token, domainInit(domainID, { method: 'GET' }))
  return (data.conversations ?? []).map(mapConversation)
}

export async function getConversation(
  token: string,
  domainID: string,
  id: string,
): Promise<{ conversation: Conversation; messages: ChatMessage[] }> {
  const data = await apiJSON<ConversationDetailDTO>(`/api/v1/conversations/${id}`, token, domainInit(domainID, { method: 'GET' }))
  const messages: ChatMessage[] = (data.messages ?? []).map((m, i) => ({
    id: i,
    role: m.role as 'user' | 'assistant',
    content: m.content,
  }))
  return { conversation: mapConversation(data.conversation), messages }
}

export async function deleteConversation(token: string, domainID: string, id: string): Promise<void> {
  const res = await fetch(buildURL(`/api/v1/conversations/${id}`), {
    method: 'DELETE',
    headers: authHeaders(token, domainID),
  })
  if (!res.ok) {
    throw new Error(await readError(res))
  }
}

export async function retrieveChat(
  token: string,
  query: string,
  recordIDs: string[],
  sourceIDs: string[] = [],
  topK = 5,
): Promise<ChatMatch[]> {
  const data = await apiJSON<ChatResponseDTO>('/api/v1/chat', token, {
    method: 'POST',
    body: JSON.stringify({
      query,
      record_ids: recordIDs,
      source_ids: sourceIDs,
      top_k: topK,
    }),
  })

  return data.matches.map(m => ({
    chunkID: m.chunk_id,
    recordID: m.record_id,
    recordName: m.record_name,
    recordFormat: toRecordFormat(m.record_format),
    sourceID: m.source_id,
    sourceName: m.source_name,
    chunkIndex: m.chunk_index,
    pageNumber: m.page_number,
    content: m.content,
  }))
}
