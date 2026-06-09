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
}

export interface Conversation {
  id: string
  title: string
  createdAt: string
  updatedAt: string
}

export type SourceType = 'google_drive' | 'rclone'

export interface DriveSourceDraft {
  sourceType: SourceType
  name: string
  rcloneRemote?: string
  rcloneRootPath?: string
  rcloneScopePaths?: string[]
  selectedRclonePaths?: string[]
  rcloneConfigRef?: string
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
  rcloneRemote?: string
  rcloneRootPath?: string
  rcloneScopePaths?: string[]
  selectedRclonePaths?: string[]
  rcloneConfigRef?: string
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
