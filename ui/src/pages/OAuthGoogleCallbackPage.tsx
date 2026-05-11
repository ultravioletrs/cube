import { useEffect } from 'react'

export default function OAuthGoogleCallbackPage() {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code') ?? ''
    const state = params.get('state') ?? ''
    const error = params.get('error') ?? ''

    if (window.opener && !window.opener.closed) {
      window.opener.postMessage({
        type: 'google_oauth_callback',
        code,
        state,
        error,
      }, window.location.origin)
    }
    window.close()
  }, [])

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', fontFamily: 'Space Grotesk, sans-serif', color: 'var(--text)' }}>
      Finishing Google sign-in...
    </div>
  )
}

