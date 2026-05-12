import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react'
import type { ReactNode } from 'react'
import { magistralaAuthService } from '@/lib/auth/magistrala-service'
import type { AuthService } from '@/lib/auth/service'
import type { AuthTokens, AuthUser } from '@/lib/auth/types'

const REFRESH_TOKEN_KEY = 'cube_refresh_token'
const ACCESS_TOKEN_KEY = 'cube_access_token'
const CHAT_KEY = 'cube_chat'
const AUTO_REFRESH_FALLBACK_MS = 45 * 60 * 1000
const AUTO_REFRESH_SKEW_MS = 60 * 1000
const AUTO_REFRESH_MIN_DELAY_MS = 5 * 1000
let inFlightSessionRestore: Promise<{ user: AuthUser; tokens: AuthTokens } | null> | null = null

function readStoredTokens(): AuthTokens | null {
  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY) ?? ''
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY) ?? ''
  if (!accessToken && !refreshToken) return null
  return { accessToken, refreshToken }
}

function persistTokens(tokens: AuthTokens): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken)
}

function clearStoredTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(CHAT_KEY)
}

function decodeBase64URL(input: string): string {
  const normalized = input.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(normalized.length + ((4 - (normalized.length % 4)) % 4), '=')
  return atob(padded)
}

function tokenExpiryMs(token: string): number | null {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return null
    const payload = JSON.parse(decodeBase64URL(parts[1])) as { exp?: unknown }
    if (typeof payload.exp !== 'number') return null
    return payload.exp * 1000
  } catch {
    return null
  }
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
  service = magistralaAuthService,
}: AuthProviderProps) {
  const initialStoredTokens = readStoredTokens()
  // Initialise isLoading from localStorage to avoid a flicker on first render.
  const [state, setState] = useState<AuthState>(() => ({
    user: null,
    tokens: null,
    isLoading: initialStoredTokens !== null,
    isAuthenticated: false,
  }))
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const refreshInFlightRef = useRef(false)
  const sessionVersionRef = useRef(0)

  const clearRefreshTimer = useCallback(() => {
    if (!refreshTimerRef.current) return
    clearTimeout(refreshTimerRef.current)
    refreshTimerRef.current = null
  }, [])

  useEffect(() => {
    const stored = readStoredTokens()
    if (!stored) return

    let cancelled = false
    const restoreVersion = sessionVersionRef.current

    if (!inFlightSessionRestore) {
      inFlightSessionRestore = (stored.refreshToken
        ? service
            .refreshTokens(stored.refreshToken)
            .then(async refreshed => {
              persistTokens(refreshed)
              const user = await service.getProfile(refreshed.accessToken)
              return { user, tokens: refreshed }
            })
        : Promise.reject(new Error('missing refresh token')))
        .catch(async () => {
          if (!stored.accessToken) return null
          try {
            const user = await service.getProfile(stored.accessToken)
            const fallbackTokens: AuthTokens = {
              accessToken: stored.accessToken,
              refreshToken: stored.refreshToken,
            }
            persistTokens(fallbackTokens)
            return { user, tokens: fallbackTokens }
          } catch {
            clearStoredTokens()
            return null
          }
        })
        .finally(() => {
          inFlightSessionRestore = null
        })
    }

    inFlightSessionRestore.then(result => {
      if (cancelled || sessionVersionRef.current !== restoreVersion) return
      if (result) {
        setState({ user: result.user, tokens: result.tokens, isLoading: false, isAuthenticated: true })
        return
      }
      setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
    })

    return () => {
      cancelled = true
    }
  }, [service])

  useEffect(() => {
    clearRefreshTimer()
    if (!state.isAuthenticated || !state.tokens?.refreshToken) return

    const accessToken = state.tokens.accessToken
    const refreshToken = state.tokens.refreshToken
    const sessionVersion = sessionVersionRef.current
    const expiresAt = tokenExpiryMs(accessToken)
    const now = Date.now()
    const delay = expiresAt
      ? Math.max(AUTO_REFRESH_MIN_DELAY_MS, expiresAt - AUTO_REFRESH_SKEW_MS - now)
      : AUTO_REFRESH_FALLBACK_MS

    refreshTimerRef.current = setTimeout(() => {
      if (refreshInFlightRef.current) return
      refreshInFlightRef.current = true

      void service
        .refreshTokens(refreshToken)
        .then(async refreshed => {
          if (sessionVersionRef.current !== sessionVersion) return
          const user = await service.getProfile(refreshed.accessToken)
          persistTokens(refreshed)
          setState({ user, tokens: refreshed, isLoading: false, isAuthenticated: true })
        })
        .catch(() => {
          if (sessionVersionRef.current !== sessionVersion) return
          clearStoredTokens()
          setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
        })
        .finally(() => {
          refreshInFlightRef.current = false
        })
    }, delay)

    return clearRefreshTimer
  }, [clearRefreshTimer, service, state.isAuthenticated, state.tokens])

  useEffect(() => clearRefreshTimer, [clearRefreshTimer])

  const login = useCallback(
    async (identity: string, password: string) => {
      sessionVersionRef.current += 1
      const tokens = await service.login({ identity, password })
      const user = await service.getProfile(tokens.accessToken)
      persistTokens(tokens)
      setState({ user, tokens, isLoading: false, isAuthenticated: true })
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
    const token = state.tokens?.accessToken
    if (token) void service.logout(token)
    sessionVersionRef.current += 1
    clearRefreshTimer()
    clearStoredTokens()
    setState({ user: null, tokens: null, isLoading: false, isAuthenticated: false })
  }, [clearRefreshTimer, service, state.tokens?.accessToken])

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
