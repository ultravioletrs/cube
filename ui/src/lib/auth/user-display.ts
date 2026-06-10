// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthUser } from './types'

export interface UserMetadataRow {
  label: string
  value: string
}

function clean(value?: string): string {
  return (value ?? '').trim()
}

function isOpaqueUserValue(user: AuthUser, value: string): boolean {
  return value === clean(user.id)
}

export function userEmail(user: AuthUser): string {
  const email = clean(user.email)
  if (!email || !email.includes('@') || isOpaqueUserValue(user, email)) return ''
  return email
}

export function userUsername(user: AuthUser): string {
  const username = clean(user.username)
  if (!username || isOpaqueUserValue(user, username)) return ''
  return username
}

export function userDisplayName(user: AuthUser): string {
  const parts = [clean(user.firstName), clean(user.lastName)].filter(Boolean)
  if (parts.length > 0) return parts.join(' ')

  const username = userUsername(user)
  if (username) return username

  const email = userEmail(user)
  if (email) return email

  return 'User'
}

export function userInitials(user: AuthUser): string {
  const firstName = clean(user.firstName)
  const lastName = clean(user.lastName)
  if (firstName && lastName) return `${firstName[0]}${lastName[0]}`.toUpperCase()

  const display = userDisplayName(user)
  if (display === 'User') return 'U'

  const localPart = display.includes('@') ? display.split('@')[0] : display
  const segments = localPart.split(/[^A-Za-z0-9]+/).filter(Boolean)
  if (segments.length >= 2) return `${segments[0][0]}${segments[1][0]}`.toUpperCase()

  const compact = localPart.replace(/[^A-Za-z0-9]/g, '')
  return (compact.slice(0, 2) || 'U').toUpperCase()
}

export function userMetadataRows(user: AuthUser): UserMetadataRow[] {
  const rows: UserMetadataRow[] = []
  const email = userEmail(user)
  const username = userUsername(user)
  if (email) rows.push({ label: 'EMAIL', value: email })
  if (username) rows.push({ label: 'USERNAME', value: username })
  if (user.role) rows.push({ label: 'ROLE', value: user.role })
  return rows
}
