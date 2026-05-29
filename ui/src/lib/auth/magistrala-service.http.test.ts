// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { afterEach, describe, expect, it, vi } from 'vitest'

import { magistralaAuthService } from './magistrala-service'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('magistralaAuthService HTTP handling', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('posts login credentials to the users token endpoint', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse(
        {
          access_token: 'access-123',
          refresh_token: 'refresh-123',
        },
        201,
      ),
    )

    await expect(
      magistralaAuthService.login({
        identity: 'jane@example.com',
        password: 'secret123',
      }),
    ).resolves.toEqual({
      accessToken: 'access-123',
      refreshToken: 'refresh-123',
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith(
      `${window.location.origin}/users/tokens/issue`,
      expect.objectContaining({
        method: 'POST',
        credentials: 'same-origin',
        body: JSON.stringify({
          identity: 'jane@example.com',
          secret: 'secret123',
          username: 'jane@example.com',
          password: 'secret123',
        }),
        headers: expect.objectContaining({
          Accept: 'application/json',
          'Content-Type': 'application/json',
        }),
      }),
    )
  })

  it('returns a clear message when proxy responds with empty non-JSON body', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('', {
        status: 502,
        statusText: 'Bad Gateway',
        headers: { 'Content-Type': 'text/plain' },
      }),
    )

    await expect(
      magistralaAuthService.login({
        identity: 'jane@example.com',
        password: 'secret123',
      }),
    ).rejects.toThrow('Authentication service is unavailable. Please try again.')
  })

  it('maps backend password mismatch to a user-friendly error', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse({ message: 'compare hash and password failed' }, 422),
    )

    await expect(
      magistralaAuthService.login({
        identity: 'admin',
        password: 'wrongpass',
      }),
    ).rejects.toThrow('Invalid username or password.')
  })

  it('does not crash on malformed JSON error payloads', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('{', {
        status: 502,
        statusText: 'Bad Gateway',
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    await expect(
      magistralaAuthService.login({
        identity: 'admin',
        password: 'wrongpass',
      }),
    ).rejects.toThrow('Authentication service is unavailable. Please try again.')
  })
})
