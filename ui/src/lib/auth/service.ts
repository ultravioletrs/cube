// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthTokens, AuthUser, LoginCredentials, RegisterCredentials } from './types'

export interface AuthSession {
  user: AuthUser
  tokens: AuthTokens
  expiresAt?: string
}

export interface AuthService {
  login(credentials: LoginCredentials): Promise<AuthSession>
  getSession(): Promise<AuthSession | null>
  register(credentials: RegisterCredentials): Promise<AuthUser>
  logout(): Promise<void>
}
