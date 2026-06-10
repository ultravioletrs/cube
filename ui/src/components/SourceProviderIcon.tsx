// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type React from 'react'
import type { SourceType } from '@/types'

interface SourceProviderIconProps {
  sourceType: SourceType
  framed?: boolean
  size?: number
}

const providerStyles: Record<SourceType, { bg: string; border: string }> = {
  google_drive: {
    bg: 'rgba(66,133,244,0.12)',
    border: 'rgba(66,133,244,0.2)',
  },
  s3: {
    bg: 'rgba(255,153,0,0.12)',
    border: 'rgba(255,153,0,0.26)',
  },
  microsoft: {
    bg: 'rgba(0,120,212,0.12)',
    border: 'rgba(0,120,212,0.25)',
  },
}

function GoogleDriveMark({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M8.5 3H15.5L22 13H16L13 8.5L10 13H2L8.5 3Z" fill="#4285f4" opacity="0.85" />
      <path d="M2 13L5.5 19H18.5L22 13H16L13 18H11L8 13H2Z" fill="#34a853" opacity="0.85" />
      <path d="M10 13L13 8.5L16 13H10Z" fill="#fbbc04" />
    </svg>
  )
}

function S3Mark({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M6 6.5C6 4.84 8.69 3.5 12 3.5C15.31 3.5 18 4.84 18 6.5V17.5C18 19.16 15.31 20.5 12 20.5C8.69 20.5 6 19.16 6 17.5V6.5Z" fill="#ff9900" opacity="0.18" />
      <path d="M18 6.5C18 8.16 15.31 9.5 12 9.5C8.69 9.5 6 8.16 6 6.5M18 6.5C18 4.84 15.31 3.5 12 3.5C8.69 3.5 6 4.84 6 6.5M18 6.5V17.5C18 19.16 15.31 20.5 12 20.5C8.69 20.5 6 19.16 6 17.5V6.5M18 12C18 13.66 15.31 15 12 15C8.69 15 6 13.66 6 12" stroke="#ff9900" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M9.2 12.2H14.8M9.2 16.7H14.8" stroke="#ffb84d" strokeWidth="1.4" strokeLinecap="round" />
    </svg>
  )
}

function MicrosoftDriveMark({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M8.4 17.5H17.1C19.1 17.5 20.75 15.9 20.75 13.94C20.75 12.1 19.35 10.57 17.55 10.39C16.89 7.75 14.48 5.8 11.63 5.8C9.14 5.8 6.99 7.29 6.09 9.42C4.47 9.78 3.25 11.21 3.25 12.92C3.25 14.98 4.95 16.65 7.06 16.65H8.4" fill="#0078d4" opacity="0.16" />
      <path d="M8.4 17.5H17.1C19.1 17.5 20.75 15.9 20.75 13.94C20.75 12.1 19.35 10.57 17.55 10.39C16.89 7.75 14.48 5.8 11.63 5.8C9.14 5.8 6.99 7.29 6.09 9.42C4.47 9.78 3.25 11.21 3.25 12.92C3.25 14.98 4.95 16.65 7.06 16.65H8.4" stroke="#28a8ea" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M10.2 13.55L12.2 11.52L14.2 13.55L12.2 15.58L10.2 13.55Z" fill="#0078d4" />
      <path d="M12.2 11.52L15.2 8.5L18.15 11.48L15.16 14.5L12.2 11.52Z" fill="#50e6ff" opacity="0.9" />
    </svg>
  )
}

function SourceProviderMark({ sourceType, size }: { sourceType: SourceType; size: number }) {
  if (sourceType === 'google_drive') return <GoogleDriveMark size={size} />
  if (sourceType === 's3') return <S3Mark size={size} />
  return <MicrosoftDriveMark size={size} />
}

export default function SourceProviderIcon({ sourceType, framed = false, size = 18 }: SourceProviderIconProps) {
  if (!framed) return <SourceProviderMark sourceType={sourceType} size={size} />

  const colors = providerStyles[sourceType]
  const frameStyle: React.CSSProperties = {
    width: '36px',
    height: '36px',
    borderRadius: '8px',
    background: colors.bg,
    border: `1px solid ${colors.border}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  }

  return (
    <div style={frameStyle}>
      <SourceProviderMark sourceType={sourceType} size={size} />
    </div>
  )
}
