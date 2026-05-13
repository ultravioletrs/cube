// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthService } from './service'
import type { AuthTokens, AuthUser } from './types'
import type { LoginCredentials, RegisterCredentials } from './types'

interface RawTokens {
  access_token: string
  refresh_token: string
}

interface RawUser {
  id: string
  email: string
  username?: string
  credentials?: { username?: string; secret?: string }
  first_name?: string
  last_name?: string
  role?: string
  created_at?: string
}

interface ParsedResponse {
  data: unknown | null
  text: string
  parseFailed: boolean
}

const RETRYABLE_ERROR_PATTERNS = [
  /socket hang up/i,
  /context canceled/i,
  /status code:canceled/i,
  /networkerror/i,
  /fetch failed/i,
  /ecconnreset/i,
  /timed out/i,
  /service unavailable/i,
]
const RETRY_BASE_DELAY_MS = 300

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

function isRetryableError(err: unknown): boolean {
  if (!(err instanceof Error)) return false
  return RETRYABLE_ERROR_PATTERNS.some(pattern => pattern.test(err.message))
}

async function withRetry<T>(attempts: number, fn: () => Promise<T>): Promise<T> {
  let lastErr: unknown
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      return await fn()
    } catch (err) {
      lastErr = err
      if (attempt === attempts - 1 || !isRetryableError(err)) {
        throw err
      }
      await sleep(RETRY_BASE_DELAY_MS * (attempt + 1))
    }
  }
  throw lastErr instanceof Error ? lastErr : new Error('Authentication request failed.')
}

function mapTokens(raw: RawTokens): AuthTokens {
  if (!raw.access_token || !raw.refresh_token) {
    throw new Error('Authentication service returned incomplete token data.')
  }

  return {
    accessToken: raw.access_token,
    refreshToken: raw.refresh_token,
  }
}

function mapUser(raw: RawUser): AuthUser {
  if (!raw.id || !raw.email) {
    throw new Error('Authentication service returned incomplete user profile data.')
  }

  return {
    id: raw.id,
    email: raw.email,
    username: raw.username ?? raw.credentials?.username ?? raw.email,
    firstName: raw.first_name,
    lastName: raw.last_name,
    role: raw.role,
    createdAt: raw.created_at,
  }
}

function usersURL(path: string): string {
  // Default to current origin so /users/* goes through the Vite dev proxy,
  // avoiding CORS. Set VITE_MAGISTRALA_USERS_URL in production.
  const base = import.meta.env.VITE_MAGISTRALA_USERS_URL ?? window.location.origin
  return new URL(path, base).toString()
}

function asObject(value: unknown): Record<string, unknown> | null {
  if (typeof value !== 'object' || value === null) return null
  return value as Record<string, unknown>
}

async function parseResponse(response: Response): Promise<ParsedResponse> {
  const text = await response.text()
  if (!text) return { data: null, text: '', parseFailed: false }

  const contentType = response.headers.get('content-type')?.toLowerCase() ?? ''
  const maybeJSON = contentType.includes('application/json') || text.trimStart().startsWith('{') || text.trimStart().startsWith('[')
  if (!maybeJSON) return { data: null, text, parseFailed: false }

  try {
    return { data: JSON.parse(text), text, parseFailed: false }
  } catch {
    return { data: null, text, parseFailed: true }
  }
}

function responseErrorMessage(response: Response, parsed: ParsedResponse): string {
  const data = asObject(parsed.data)
  const message = data?.['message']
  if (typeof message === 'string' && message.trim()) return message

  if (!parsed.parseFailed) {
    const text = parsed.text.trim()
    if (text) return text
  }

  if (response.status >= 500) {
    return 'Authentication service is unavailable. Please try again.'
  }

  return `Authentication request failed (${response.status}).`
}

async function requestUsers<T>(
  path: string,
  options: {
    method: 'GET' | 'POST'
    token?: string
    body?: unknown
  },
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }

  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }

  const response = await fetch(usersURL(path), {
    method: options.method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    credentials: 'same-origin',
  })

  const parsed = await parseResponse(response)
  if (!response.ok) {
    throw new Error(responseErrorMessage(response, parsed))
  }

  if (parsed.data === null) {
    throw new Error('Authentication service returned an empty response.')
  }

  return parsed.data as T
}

export const magistralaAuthService: AuthService = {
  async login({ identity, password }: LoginCredentials): Promise<AuthTokens> {
    const raw = await withRetry(3, () => requestUsers<RawTokens>('/users/tokens/issue', {
      method: 'POST',
      // Send both payload shapes to support newer and older Magistrala versions.
      body: {
        identity,
        secret: password,
        username: identity,
        password,
      },
    }))
    return mapTokens(raw)
  },

  async register({ email, username, password, firstName, lastName }: RegisterCredentials): Promise<AuthUser> {
    const raw = await requestUsers<RawUser>('/users', {
      method: 'POST',
      body: {
        email,
        first_name: firstName,
        last_name: lastName,
        credentials: { username, secret: password },
      },
    })
    return mapUser(raw)
  },

  async refreshTokens(refreshToken: string): Promise<AuthTokens> {
    if (!refreshToken.trim()) {
      throw new Error('Refresh token is missing.')
    }
    const raw = await withRetry(3, () => requestUsers<RawTokens>('/users/tokens/refresh', {
      method: 'POST',
      token: refreshToken,
      body: {},
    }))
    return mapTokens(raw)
  },

  async getProfile(accessToken: string): Promise<AuthUser> {
    const raw = await requestUsers<RawUser>('/users/profile', {
      method: 'GET',
      token: accessToken,
    })
    return mapUser(raw)
  },

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async logout(_accessToken: string): Promise<void> {
    // SDK has no revocation endpoint; access tokens expire after 1h.
  },
}
