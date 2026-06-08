// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { useState } from 'react'
import { useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'

const inputStyle: React.CSSProperties = {
  width: '100%',
  background: 'rgba(255,255,255,0.04)',
  border: '1px solid var(--border)',
  borderRadius: '8px',
  padding: '9px 12px',
  color: 'var(--text)',
  fontFamily: 'Space Grotesk, sans-serif',
  fontSize: '14px',
  outline: 'none',
  boxSizing: 'border-box',
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontFamily: 'Space Grotesk, sans-serif',
  fontWeight: '600',
  fontSize: '13px',
  color: 'var(--text)',
  marginBottom: '6px',
}

export default function AuthPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const { login, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/dashboard'
  const atomUIURL = import.meta.env.VITE_ATOM_UI_URL ?? '/atom'

  // Auth page should always redirect authenticated users to app.
  if (isAuthenticated) return <Navigate to={from} replace />

  async function handleSubmit(e: React.SyntheticEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      await login(email, password)
      navigate(from)
    } catch (err) {
      setError(extractMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)' }}>
      <div style={{ width: '100%', maxWidth: '380px', background: 'var(--card-bg)', border: '1px solid var(--border)', borderRadius: '16px', padding: '32px', boxShadow: '0 24px 80px rgba(0,0,0,0.4)' }}>
        <div style={{ marginBottom: '24px' }}>
          <h1 style={{ fontFamily: 'Space Grotesk, sans-serif', fontWeight: '700', fontSize: '22px', color: 'var(--text)', margin: '0 0 6px', letterSpacing: '-0.02em' }}>
            Welcome back
          </h1>
          <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>
            Sign in to your Cube AI account
          </p>
        </div>

        <form style={{ display: 'flex', flexDirection: 'column', gap: '16px' }} onSubmit={handleSubmit}>
          <div>
            <label style={labelStyle} htmlFor="email">Email or username</label>
            <input
              id="email"
              type="text"
              placeholder="you@example.com or johndoe"
              style={inputStyle}
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <label style={labelStyle} htmlFor="password">Password</label>
            <div style={{ position: 'relative' }}>
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                placeholder="••••••••"
                style={{ ...inputStyle, paddingRight: '40px' }}
                value={password}
                onChange={e => setPassword(e.target.value)}
                required
                minLength={8}
              />
              <button
                type="button"
                onClick={() => setShowPassword(v => !v)}
                style={{ position: 'absolute', right: '10px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', padding: '2px', color: 'var(--text-muted)', display: 'flex', alignItems: 'center' }}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
                    <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
                    <line x1="1" y1="1" x2="23" y2="23"/>
                  </svg>
                ) : (
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                    <circle cx="12" cy="12" r="3"/>
                  </svg>
                )}
              </button>
            </div>
          </div>

          {error && (
            <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: '#ff6b6b', margin: 0, padding: '8px 12px', background: 'rgba(255,107,107,0.08)', borderRadius: '6px', border: '1px solid rgba(255,107,107,0.2)' }}>
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            style={{ background: loading ? 'rgba(0,212,180,0.5)' : 'var(--accent)', border: 'none', color: '#070c16', padding: '10px 16px', borderRadius: '8px', cursor: loading ? 'not-allowed' : 'pointer', fontFamily: 'Space Grotesk, sans-serif', fontSize: '14px', fontWeight: '700', marginTop: '4px' }}
          >
            {loading ? 'Please wait…' : 'Sign in'}
          </button>
        </form>

        <p style={{ fontFamily: 'Space Grotesk, sans-serif', fontSize: '13px', color: 'var(--text-muted)', textAlign: 'center', marginTop: '20px', marginBottom: 0 }}>
          <a
            href={atomUIURL}
            target="_blank"
            rel="noreferrer"
            style={{ color: 'var(--text)', fontWeight: '600', textDecoration: 'underline', textUnderlineOffset: '3px' }}
          >
            Identity & Access
          </a>
        </p>
      </div>
    </div>
  )
}

function extractMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  return 'Something went wrong. Please try again.'
}
