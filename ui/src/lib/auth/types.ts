// Domain types for authentication.
// These are Veda's own types — no SDK or backend types leak above this file.

export interface AuthUser {
  id: string
  email: string
  username: string
  firstName?: string
  lastName?: string
  role?: string
  createdAt?: string
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}

export interface LoginCredentials {
  /** Email address or username */
  identity: string
  password: string
}

export interface RegisterCredentials {
  email: string
  username: string
  password: string
  firstName?: string
  lastName?: string
}
