// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { DriveFileOption } from '@/lib/embedder/service'

export interface DriveFileTypeVisual {
  badge: string
  label: string
  iconBg: string
  iconBorder: string
  iconColor: string
}

function extensionFromName(name: string): string {
  const idx = name.lastIndexOf('.')
  if (idx < 0 || idx === name.length - 1) return ''
  return name.slice(idx + 1).toLowerCase()
}

function shortBadge(ext: string, fallback: string): string {
  if (!ext) return fallback
  return ext.slice(0, 4).toUpperCase()
}

export function resolveDriveFileTypeVisual(file: Pick<DriveFileOption, 'name' | 'mimeType'>): DriveFileTypeVisual {
  const mime = (file.mimeType || '').toLowerCase()
  const ext = extensionFromName(file.name)

  if (mime.includes('pdf') || ext === 'pdf') {
    return { badge: 'PDF', label: 'Document', iconBg: 'rgba(255,80,80,0.15)', iconBorder: 'rgba(255,80,80,0.35)', iconColor: '#ff7b7b' }
  }

  if (
    mime.startsWith('image/') ||
    ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'tiff', 'ico'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'IMG'), label: 'Image', iconBg: 'rgba(66,133,244,0.15)', iconBorder: 'rgba(66,133,244,0.35)', iconColor: '#7caeff' }
  }

  if (
    mime.includes('sheet') ||
    mime.includes('spreadsheet') ||
    ['xls', 'xlsx', 'csv', 'tsv'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'SHEET'), label: 'Sheet', iconBg: 'rgba(52,168,83,0.15)', iconBorder: 'rgba(52,168,83,0.35)', iconColor: '#79d996' }
  }

  if (
    mime.includes('presentation') ||
    ['ppt', 'pptx', 'key'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'PPT'), label: 'Slides', iconBg: 'rgba(251,188,4,0.15)', iconBorder: 'rgba(251,188,4,0.4)', iconColor: '#ffd86a' }
  }

  if (
    mime.includes('wordprocessingml') ||
    mime.includes('msword') ||
    mime.includes('google-apps.document') ||
    ['doc', 'docx', 'odt', 'rtf'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'DOC'), label: 'Text doc', iconBg: 'rgba(66,133,244,0.15)', iconBorder: 'rgba(66,133,244,0.35)', iconColor: '#7caeff' }
  }

  if (
    mime.includes('text') ||
    mime.includes('json') ||
    mime.includes('xml') ||
    ['txt', 'md', 'json', 'xml', 'yml', 'yaml', 'ini', 'log'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'TXT'), label: 'Text', iconBg: 'rgba(148,163,184,0.15)', iconBorder: 'rgba(148,163,184,0.35)', iconColor: '#cbd5e1' }
  }

  if (
    mime.includes('zip') ||
    mime.includes('compressed') ||
    ['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(ext)
  ) {
    return { badge: shortBadge(ext, 'ZIP'), label: 'Archive', iconBg: 'rgba(245,158,11,0.15)', iconBorder: 'rgba(245,158,11,0.35)', iconColor: '#f9c46b' }
  }

  return { badge: shortBadge(ext, 'FILE'), label: 'File', iconBg: 'rgba(148,163,184,0.14)', iconBorder: 'rgba(148,163,184,0.32)', iconColor: '#cbd5e1' }
}
