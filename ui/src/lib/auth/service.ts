// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthTokens, AuthUser, LoginCredentials, RegisterCredentials } from './types'

export interface AuthSession {
  user: AuthUser
  tokens: AuthTokens
  expiresAt?: string
}

export interface RegisterResult {
  user: AuthUser
  /** When true, the account exists but must verify its email before it can sign in. */
  verificationRequired: boolean
}

export interface AuthService {
  login(credentials: LoginCredentials): Promise<AuthSession>
  getSession(): Promise<AuthSession | null>
  register(credentials: RegisterCredentials): Promise<RegisterResult>
  logout(): Promise<void>
}
