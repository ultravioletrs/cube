// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthService, AuthSession, RegisterResult } from './service'
import type { AuthTokens, AuthUser, LoginCredentials, RegisterCredentials } from './types'

export const ATOM_SESSION_STORAGE_KEY = 'cube_atom_session'

// Well-known UUID of the seeded ATOM admin entity (atom::config::ADMIN_ENTITY_ID).
// Identity & Access is an admin-only surface, so we treat this entity as admin.
const ADMIN_ENTITY_ID = '00000000-0000-0000-0000-000000000001'

interface StoredAtomSession {
  accessToken: string
  entityId: string
  sessionId: string
  expiresAt: string
  identity?: string
}

interface GraphQLError {
  message?: string
}

interface GraphQLResponse<T> {
  data?: T
  errors?: GraphQLError[]
}

interface LoginData {
  login: {
    token: string
    entityId: string
    sessionId: string
    expiresAt: string
  }
}

interface SessionData {
  session: {
    id: string
    entityId: string
    expiresAt: string
    revokedAt?: string | null
  }
}

interface LogoutData {
  logout: boolean
}

interface SignupData {
  signup: {
    entityId: string
    email: string
    verificationRequired: boolean
  }
}

const atomAPIURL = import.meta.env.VITE_ATOM_API_URL ?? 'http://localhost:8080'
const atomGraphQLURL = import.meta.env.VITE_ATOM_GRAPHQL_URL ?? new URL('/graphql', atomAPIURL).toString()

const LOGIN_MUTATION = `
  mutation CubeLogin($input: LoginInput!) {
    login(input: $input) {
      token
      entityId
      sessionId
      expiresAt
    }
  }
`

const SESSION_QUERY = `
  query CubeSession($id: ID!) {
    session(id: $id) {
      id
      entityId
      expiresAt
      revokedAt
    }
  }
`

const LOGOUT_MUTATION = `
  mutation CubeLogout {
    logout
  }
`

const SIGNUP_MUTATION = `
  mutation CubeSignup($input: SignupInput!) {
    signup(input: $input) {
      entityId
      email
      verificationRequired
    }
  }
`

function isStorageAvailable(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined'
}

function isExpired(expiresAt: string): boolean {
  const expires = new Date(expiresAt).getTime()
  return Number.isNaN(expires) || expires <= Date.now()
}

function authUser(entityID: string, identity = ''): AuthUser {
  const cleanIdentity = identity.trim()
  const isEmail = cleanIdentity.includes('@')
  return {
    id: entityID,
    email: isEmail ? cleanIdentity : '',
    username: isEmail ? cleanIdentity.split('@')[0] : cleanIdentity,
    role: entityID === ADMIN_ENTITY_ID ? 'admin' : 'user',
  }
}

function authTokens(accessToken: string): AuthTokens {
  return {
    accessToken,
    refreshToken: '',
  }
}

function toSession(stored: StoredAtomSession): AuthSession {
  return {
    user: authUser(stored.entityId, stored.identity),
    tokens: authTokens(stored.accessToken),
    expiresAt: stored.expiresAt,
  }
}

function isTransientValidationError(error: unknown): boolean {
  if (error instanceof TypeError) return true
  if (error instanceof Error && error.name === 'AbortError') return true
  if (typeof error === 'object' && error !== null && 'name' in error) {
    return (error as { name?: unknown }).name === 'AbortError'
  }
  return false
}

function parseStoredSession(value: string | null): StoredAtomSession | null {
  if (!value) return null
  try {
    const parsed = JSON.parse(value) as Partial<StoredAtomSession>
    if (
      typeof parsed.accessToken !== 'string'
      || typeof parsed.entityId !== 'string'
      || typeof parsed.sessionId !== 'string'
      || typeof parsed.expiresAt !== 'string'
      || !parsed.accessToken
      || !parsed.entityId
      || !parsed.sessionId
      || !parsed.expiresAt
    ) {
      return null
    }
    if (isExpired(parsed.expiresAt)) return null
    return {
      accessToken: parsed.accessToken,
      entityId: parsed.entityId,
      sessionId: parsed.sessionId,
      expiresAt: parsed.expiresAt,
      identity: typeof parsed.identity === 'string' ? parsed.identity.trim() : undefined,
    }
  } catch {
    return null
  }
}

export function loadStoredAtomSession(): StoredAtomSession | null {
  if (!isStorageAvailable()) return null
  return parseStoredSession(window.localStorage.getItem(ATOM_SESSION_STORAGE_KEY))
}

export function getStoredAtomAccessToken(): string | null {
  return loadStoredAtomSession()?.accessToken ?? null
}

export function clearStoredAtomSession(): void {
  if (!isStorageAvailable()) return
  window.localStorage.removeItem(ATOM_SESSION_STORAGE_KEY)
}

function storeAtomSession(session: StoredAtomSession): void {
  if (!isStorageAvailable()) return
  window.localStorage.setItem(ATOM_SESSION_STORAGE_KEY, JSON.stringify(session))
}

async function graphQL<T>(
  query: string,
  variables?: Record<string, unknown>,
  accessToken?: string,
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    'Content-Type': 'application/json',
  }
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`

  const response = await fetch(atomGraphQLURL, {
    method: 'POST',
    credentials: 'omit',
    headers,
    body: JSON.stringify({ query, variables }),
  })

  const text = await response.text()
  const payload = text ? JSON.parse(text) as GraphQLResponse<T> : {}
  if (!response.ok) {
    throw new Error(payload.errors?.[0]?.message ?? `ATOM GraphQL request failed (${response.status}).`)
  }
  if (payload.errors?.length) {
    throw new Error(payload.errors[0]?.message ?? 'ATOM GraphQL request failed.')
  }
  if (!payload.data) {
    throw new Error('ATOM GraphQL returned an empty response.')
  }
  return payload.data
}

async function validateStoredSession(stored: StoredAtomSession): Promise<AuthSession | null> {
  try {
    const data = await graphQL<SessionData>(
      SESSION_QUERY,
      { id: stored.sessionId },
      stored.accessToken,
    )
    if (
      data.session.revokedAt
      || data.session.entityId !== stored.entityId
      || isExpired(data.session.expiresAt)
    ) {
      clearStoredAtomSession()
      return null
    }
    const validated = {
      ...stored,
      expiresAt: data.session.expiresAt,
    }
    storeAtomSession(validated)
    return toSession(validated)
  } catch (error) {
    if (isTransientValidationError(error)) return toSession(stored)
    return null
  }
}

export const atomAuthService: AuthService = {
  async login({ identity, password }: LoginCredentials): Promise<AuthSession> {
    const data = await graphQL<LoginData>(LOGIN_MUTATION, {
      input: {
        identifier: identity,
        secret: password,
        kind: 'password',
      },
    })
    const session = {
      accessToken: data.login.token,
      entityId: data.login.entityId,
      sessionId: data.login.sessionId,
      expiresAt: data.login.expiresAt,
      identity: identity.trim(),
    }
    if (
      !session.accessToken
      || !session.entityId
      || !session.sessionId
      || !session.expiresAt
      || isExpired(session.expiresAt)
    ) {
      throw new Error('ATOM returned an invalid session.')
    }
    storeAtomSession(session)
    return toSession(session)
  },

  async getSession(): Promise<AuthSession | null> {
    const stored = loadStoredAtomSession()
    if (!stored) {
      clearStoredAtomSession()
      return null
    }
    return validateStoredSession(stored)
  },

  async register({ email, username, password }: RegisterCredentials): Promise<RegisterResult> {
    const data = await graphQL<SignupData>(SIGNUP_MUTATION, {
      input: {
        name: username,
        email,
        password,
      },
    })
    if (!data.signup.entityId) {
      throw new Error('ATOM returned an invalid signup response.')
    }
    const user = authUser(data.signup.entityId)
    user.email = data.signup.email
    user.username = username
    return {
      user,
      verificationRequired: data.signup.verificationRequired,
    }
  },

  async logout(): Promise<void> {
    const token = getStoredAtomAccessToken()
    clearStoredAtomSession()
    if (!token) return
    try {
      await graphQL<LogoutData>(LOGOUT_MUTATION, undefined, token)
    } catch {
      // Local logout should still complete if the remote session is already gone.
    }
  },
}
