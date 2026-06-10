// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { Domain as SDKDomain } from '@/lib/platform/service'

export type { SDKDomain as PlatformDomain }

export interface ActiveWorkspace {
  id: string
  name: string
  route?: string
  status?: string
}

export type ActiveDomain = ActiveWorkspace

export type RecordFormat = 'text' | 'pdf' | 'md' | 'docx' | 'code' | 'image' | 'link'

export interface MsgSource {
  id: string
  recordID?: string
  doc: string
  page: number
  excerpt: string
  url?: string
}

export interface ChatDebugChunk {
  rank: number
  record_id: string
  record_name: string
  external_url?: string
  chunk_index: number
  score?: number
  preview: string
}

export interface ChatDebug {
  query: string
  top_k: number
  retrieval_enabled: boolean
  skipped_reason?: string
  record_ids?: string[]
  prompt_chunks: ChatDebugChunk[]
}

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  sources?: MsgSource[]
  debug?: ChatDebug
}

export interface AppRecord {
  id: string
  sourceID?: string
  name: string
  format: RecordFormat
  status: 'queued' | 'processing' | 'indexed' | 'failed' | 'cancelled'
  createdAt: string
  description: string
  error?: string
  chunks: number | null
  ingestTotalChunks?: number | null
  ingestIndexedChunks?: number | null
  ingestStage?: string | null
  size?: string
  pages?: number | null
  url?: string
  folderPath?: string
  folderID?: string
}

export interface Conversation {
  id: string
  title: string
  createdAt: string
  updatedAt: string
}

export type SourceType = 'google_drive' | 's3' | 'microsoft'

// S3Config holds native S3/MinIO connection + scope fields.
export interface S3Config {
  endpoint?: string
  region?: string
  bucket?: string
  accessKeyID?: string
  secretAccessKey?: string
  sessionToken?: string
  useSSL?: boolean
  pathStyle?: boolean
  rootPath?: string
  scopePaths?: string[]
  selectedPaths?: string[]
}

// MicrosoftConfig holds native OneDrive/SharePoint (Graph) connection + scope fields.
export interface MicrosoftConfig {
  tenantID?: string
  clientID?: string
  clientSecret?: string
  accessToken?: string
  refreshToken?: string
  driveID?: string
  siteID?: string
  rootPath?: string
  scopePaths?: string[]
  selectedPaths?: string[]
}

export interface DriveSourceDraft {
  sourceType: SourceType
  name: string
  s3?: S3Config
  microsoft?: MicrosoftConfig
  folderLink: string
  accessToken: string
  refreshToken: string
  clientId: string
  clientSecret: string
  selectedFileIDs: string[]
  selectedFolderIDs: string[]
  syncEnabled: boolean
  autoSyncInterval: number
}

export interface DriveSource {
  id: string
  sourceType: SourceType
  name: string
  s3?: S3Config
  microsoft?: MicrosoftConfig
  folderLink: string
  accessToken: string
  refreshToken: string
  clientId: string
  clientSecret: string
  selectedFileIDs: string[]
  selectedFolderIDs: string[]
  syncEnabled: boolean
  autoSyncInterval: number
  status: 'active' | 'syncing' | 'error' | 'disconnected'
  lastSyncError?: string
  lastSyncAt?: string
  createdAt: string
}

export interface AppContext {
  records: AppRecord[]
  setRecords: React.Dispatch<React.SetStateAction<AppRecord[]>>
  driveSources: DriveSource[]
  setDriveSources: React.Dispatch<React.SetStateAction<DriveSource[]>>
  chatMessages: ChatMessage[]
  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>
  clearChatMessages: () => void
  conversationId: string | null
  setConversationId: React.Dispatch<React.SetStateAction<string | null>>
  conversations: Conversation[]
  setConversations: React.Dispatch<React.SetStateAction<Conversation[]>>
  activeWorkspace: ActiveWorkspace | null
  setActiveWorkspace: React.Dispatch<React.SetStateAction<ActiveWorkspace | null>>
  activeDomain: ActiveWorkspace | null
  setActiveDomain: React.Dispatch<React.SetStateAction<ActiveWorkspace | null>>
}
