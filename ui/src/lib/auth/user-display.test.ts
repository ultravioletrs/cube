// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { describe, expect, it } from 'vitest'

import type { AuthUser } from './types'
import { userDisplayName, userEmail, userInitials, userMetadataRows, userUsername } from './user-display'

function user(overrides: Partial<AuthUser>): AuthUser {
  return {
    id: '97f55cb7-22c9-4384-b1f6-a1dd66a1bd00',
    email: '',
    username: '',
    ...overrides,
  }
}

describe('user display helpers', () => {
  it('uses human name initials when names are present', () => {
    const u = user({ firstName: 'Jane', lastName: 'Doe', email: 'jane@example.com', username: 'jane' })

    expect(userDisplayName(u)).toBe('Jane Doe')
    expect(userInitials(u)).toBe('JD')
  })

  it('uses username when no human name exists', () => {
    const u = user({ username: 'jane.doe' })

    expect(userDisplayName(u)).toBe('jane.doe')
    expect(userInitials(u)).toBe('JD')
  })

  it('uses email when no name or username exists', () => {
    const u = user({ email: 'jane@example.com' })

    expect(userDisplayName(u)).toBe('jane@example.com')
    expect(userInitials(u)).toBe('JA')
  })

  it('does not expose opaque entity ids as display identity', () => {
    const u = user({
      email: '97f55cb7-22c9-4384-b1f6-a1dd66a1bd00',
      username: '97f55cb7-22c9-4384-b1f6-a1dd66a1bd00',
    })

    expect(userEmail(u)).toBe('')
    expect(userUsername(u)).toBe('')
    expect(userDisplayName(u)).toBe('User')
    expect(userInitials(u)).toBe('U')
    expect(userMetadataRows(u)).toEqual([])
  })
})
