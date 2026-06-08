// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  ATOM_SESSION_STORAGE_KEY,
  atomAuthService,
  getStoredAtomAccessToken,
} from './atom-auth-service'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function graphQLResponse(data: unknown, status = 200): Response {
  return jsonResponse({ data }, status)
}

function storedSession(): string {
  return JSON.stringify({
    accessToken: 'jwt-token',
    entityId: 'entity-1',
    sessionId: 'session-1',
    expiresAt: '2999-06-07T12:00:00Z',
  })
}

describe('atomAuthService', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('logs in through ATOM GraphQL and stores a JWT session', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      graphQLResponse({
        login: {
          token: 'jwt-token',
          entityId: 'entity-1',
          sessionId: 'session-1',
          expiresAt: '2999-06-07T12:00:00Z',
        },
      }),
    )

    await expect(atomAuthService.login({
      identity: 'jane@example.com',
      password: 'secret123',
    })).resolves.toMatchObject({
      user: { id: 'entity-1' },
      tokens: { accessToken: 'jwt-token', refreshToken: '' },
      expiresAt: '2999-06-07T12:00:00Z',
    })

    expect(getStoredAtomAccessToken()).toBe('jwt-token')
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/graphql', expect.objectContaining({
      method: 'POST',
      credentials: 'omit',
      body: expect.stringContaining('CubeLogin'),
    }))
  })

  it('restores and validates a stored bearer-token session', async () => {
    localStorage.setItem(ATOM_SESSION_STORAGE_KEY, storedSession())
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      graphQLResponse({
        session: {
          id: 'session-1',
          entityId: 'entity-1',
          expiresAt: '2999-06-07T12:00:00Z',
          revokedAt: null,
        },
      }),
    )

    await expect(atomAuthService.getSession()).resolves.toMatchObject({
      user: { id: 'entity-1' },
      tokens: { accessToken: 'jwt-token' },
    })
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/graphql', expect.objectContaining({
      credentials: 'omit',
      headers: expect.objectContaining({ Authorization: 'Bearer jwt-token' }),
    }))
  })

  it('clears malformed or expired sessions without calling ATOM', async () => {
    localStorage.setItem(ATOM_SESSION_STORAGE_KEY, JSON.stringify({
      accessToken: 'jwt-token',
      entityId: 'entity-1',
      sessionId: 'session-1',
      expiresAt: '2000-01-01T00:00:00Z',
    }))
    const fetchMock = vi.spyOn(globalThis, 'fetch')

    await expect(atomAuthService.getSession()).resolves.toBeNull()
    expect(localStorage.getItem(ATOM_SESSION_STORAGE_KEY)).toBeNull()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('restores stored sessions when validation is interrupted', async () => {
    const session = storedSession()
    localStorage.setItem(ATOM_SESSION_STORAGE_KEY, session)
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new DOMException('interrupted', 'AbortError'))

    await expect(atomAuthService.getSession()).resolves.toMatchObject({
      user: { id: 'entity-1' },
      tokens: { accessToken: 'jwt-token' },
    })
    expect(localStorage.getItem(ATOM_SESSION_STORAGE_KEY)).toBe(session)
  })

  it('logs out through ATOM GraphQL and clears local storage', async () => {
    localStorage.setItem(ATOM_SESSION_STORAGE_KEY, storedSession())
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(graphQLResponse({ logout: true }))

    await expect(atomAuthService.logout()).resolves.toBeUndefined()
    expect(localStorage.getItem(ATOM_SESSION_STORAGE_KEY)).toBeNull()
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/graphql', expect.objectContaining({
      method: 'POST',
      credentials: 'omit',
      headers: expect.objectContaining({ Authorization: 'Bearer jwt-token' }),
      body: expect.stringContaining('CubeLogout'),
    }))
  })
})
