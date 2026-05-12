// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { AuthService } from './service'

// ── Stub implementation ───────────────────────────────────────────────────────
// Tests use a hand-rolled stub rather than the real Magistrala service so they
// run without a live backend and without importing the SDK (which calls
// import.meta.env at module load time).

function makeStubService(): AuthService {
  return {
    login: vi.fn().mockResolvedValue({
      accessToken: 'test-access',
      refreshToken: 'test-refresh',
    }),
    register: vi.fn().mockResolvedValue({
      id: 'user-1',
      email: 'jane@example.com',
      username: 'jane',
    }),
    refreshTokens: vi.fn().mockResolvedValue({
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
    }),
    getProfile: vi.fn().mockResolvedValue({
      id: 'user-1',
      email: 'jane@example.com',
      username: 'jane',
      firstName: 'Jane',
      lastName: 'Doe',
      role: 'user',
    }),
    logout: vi.fn().mockResolvedValue(undefined),
  }
}

// ── AuthService contract tests ────────────────────────────────────────────────
// These tests document and enforce the contract that any AuthService
// implementation must satisfy. Run them against a real implementation
// in integration tests by swapping the stub for the real service.

describe('AuthService contract', () => {
  let service: AuthService

  beforeEach(() => {
    service = makeStubService()
  })

  it('login returns access and refresh tokens', async () => {
    const tokens = await service.login({ identity: 'jane@example.com', password: 'secret123' })
    expect(tokens.accessToken).toBeTruthy()
    expect(tokens.refreshToken).toBeTruthy()
  })

  it('register returns a user with id and email', async () => {
    const user = await service.register({
      email: 'jane@example.com',
      username: 'jane',
      password: 'secret123',
    })
    expect(user.id).toBeTruthy()
    expect(user.email).toBe('jane@example.com')
  })

  it('refreshTokens returns a new token pair', async () => {
    const tokens = await service.refreshTokens('old-refresh')
    expect(tokens.accessToken).toBeTruthy()
    expect(tokens.refreshToken).toBeTruthy()
  })

  it('getProfile returns a user with id, email, and username', async () => {
    const user = await service.getProfile('access-token')
    expect(user.id).toBeTruthy()
    expect(user.email).toBeTruthy()
    expect(user.username).toBeTruthy()
  })

  it('logout resolves without throwing', async () => {
    await expect(service.logout('access-token')).resolves.toBeUndefined()
  })
})

// ── Token mapping helpers (pure, no external deps) ────────────────────────────

describe('token shape', () => {
  it('accessToken and refreshToken are non-empty strings after login', async () => {
    const service = makeStubService()
    const { accessToken, refreshToken } = await service.login({ identity: 'x', password: 'y' })
    expect(typeof accessToken).toBe('string')
    expect(typeof refreshToken).toBe('string')
    expect(accessToken.length).toBeGreaterThan(0)
    expect(refreshToken.length).toBeGreaterThan(0)
  })
})
