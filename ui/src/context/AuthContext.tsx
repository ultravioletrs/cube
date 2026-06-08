// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react'
import type { ReactNode } from 'react'
import { atomAuthService, clearStoredAtomSession } from '@/lib/auth/atom-auth-service'
import type { AuthService } from '@/lib/auth/service'
import type { AuthTokens, AuthUser } from '@/lib/auth/types'

const CHAT_KEY = 'cube_chat'
const CONV_KEY = 'cube_conv_id'
const TENANT_KEY = 'cube_active_tenant'
const LEGACY_DOMAIN_KEY = 'cube_active_domain'
const LEGACY_ACCESS_TOKEN_KEY = 'cube_access_token'
const LEGACY_REFRESH_TOKEN_KEY = 'cube_refresh_token'

function clearClientSessionArtifacts(): void {
  clearStoredAtomSession()
  localStorage.removeItem(CHAT_KEY)
  localStorage.removeItem(CONV_KEY)
  localStorage.removeItem(TENANT_KEY)
  localStorage.removeItem(LEGACY_DOMAIN_KEY)
  localStorage.removeItem(LEGACY_ACCESS_TOKEN_KEY)
  localStorage.removeItem(LEGACY_REFRESH_TOKEN_KEY)
}

interface AuthState {
  user: AuthUser | null
  tokens: AuthTokens | null
  isLoading: boolean
  isAuthenticated: boolean
}

export interface AuthContextValue extends AuthState {
  login(identity: string, password: string): Promise<void>
  register(
    email: string,
    username: string,
    password: string,
    firstName?: string,
    lastName?: string,
  ): Promise<void>
  logout(): void
}

const AuthContext = createContext<AuthContextValue | null>(null)

interface AuthProviderProps {
  children: ReactNode
  service?: AuthService
}

export function AuthProvider({
  children,
  service = atomAuthService,
}: AuthProviderProps) {
  const [state, setState] = useState<AuthState>({
    user: null,
    tokens: null,
    isLoading: true,
    isAuthenticated: false,
  })

  useEffect(() => {
    let cancelled = false
    service
      .getSession()
      .then(session => {
        if (cancelled) return
        if (session) {
          setState({
            user: session.user,
            tokens: session.tokens,
            isLoading: false,
            isAuthenticated: true,
          })
          return
        }
        clearClientSessionArtifacts()
        setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
      })
      .catch(() => {
        if (cancelled) return
        clearClientSessionArtifacts()
        setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
      })

    return () => {
      cancelled = true
    }
  }, [service])

  const login = useCallback(
    async (identity: string, password: string) => {
      const session = await service.login({ identity, password })
      setState({
        user: session.user,
        tokens: session.tokens,
        isLoading: false,
        isAuthenticated: true,
      })
    },
    [service],
  )

  const register = useCallback(
    async (
      email: string,
      username: string,
      password: string,
      firstName?: string,
      lastName?: string,
    ) => {
      await service.register({ email, username, password, firstName, lastName })
      await login(email, password)
    },
    [service, login],
  )

  const logout = useCallback(() => {
    void service.logout()
    clearClientSessionArtifacts()
    setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
  }, [service])

  return (
    <AuthContext.Provider value={{ ...state, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuthContext(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuthContext must be used inside <AuthProvider>')
  return ctx
}
