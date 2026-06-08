// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

import type { AppRecord } from '@/types'

export type ImageIngestMode = 'ocr_only' | 'image_only' | 'hybrid' | 'unknown'

export function imageIngestMode(record: AppRecord): ImageIngestMode {
  if (record.format !== 'image') return 'unknown'

  const description = record.description.toLowerCase()
  if (description.includes('ocr only')) return 'ocr_only'
  if (description.includes('hybrid')) return 'hybrid'
  if (description.includes('visual embedding')) return 'image_only'
  return 'unknown'
}

export function imageIngestLabel(record: AppRecord): string {
  switch (imageIngestMode(record)) {
    case 'ocr_only':
      return 'OCR text'
    case 'hybrid':
      return 'OCR + visual'
    case 'image_only':
      return 'Visual image'
    default:
      return 'Image'
  }
}

export function imageIngestStatusText(record: AppRecord): string {
  switch (imageIngestMode(record)) {
    case 'ocr_only':
      return 'OCR text indexed · available for retrieval'
    case 'hybrid':
      return 'OCR + visual image indexed · available for retrieval'
    case 'image_only':
      return 'Visual image indexed · available for retrieval'
    default:
      return 'Image indexed · available for retrieval'
  }
}

export function imageRecordSubtext(record: AppRecord): string {
  if (record.chunks == null) return `${imageIngestLabel(record)} indexing…`
  return `${imageIngestLabel(record)} · ${record.chunks} vectors`
}
