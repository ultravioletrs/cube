// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import type { AuthTokens, AuthUser, LoginCredentials, RegisterCredentials } from './types'

export interface AuthService {
  login(credentials: LoginCredentials): Promise<AuthTokens>
  register(credentials: RegisterCredentials): Promise<AuthUser>
  refreshTokens(refreshToken: string): Promise<AuthTokens>
  getProfile(accessToken: string): Promise<AuthUser>
  logout(accessToken: string): Promise<void>
}
