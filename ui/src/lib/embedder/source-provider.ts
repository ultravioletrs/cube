// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { SourceType } from '@/types'

export function sourceProviderLabel(sourceType: SourceType): string {
  if (sourceType === 'google_drive') return 'Google Drive'
  if (sourceType === 's3') return 'S3'
  return 'OneDrive / SharePoint'
}
