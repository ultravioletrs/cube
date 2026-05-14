// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

import { CUBE_PROXY_URL } from './features'

function proxyURL(path: string): string {
  return `${CUBE_PROXY_URL}${path}`
}

// AttestationPolicy is the raw JSON stored as JSONB on the proxy.
// The schema is intentionally open — it is defined by the attestation provider.
export type AttestationPolicy = Record<string, unknown>

// getAttestationPolicy fetches the latest attestation policy for a domain.
// Endpoint: GET /{domainID}/attestation/policy
export async function getAttestationPolicy(
  token: string,
  domainID: string,
): Promise<AttestationPolicy | null> {
  const res = await fetch(proxyURL(`/${domainID}/attestation/policy`), {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`getAttestationPolicy: ${res.status} ${res.statusText}`)
  const body = await res.text()
  if (!body.trim()) return null
  return JSON.parse(body) as AttestationPolicy
}

// updateAttestationPolicy stores a new attestation policy.
// Endpoint: POST /attestation/policy
export async function updateAttestationPolicy(
  token: string,
  policy: AttestationPolicy,
): Promise<void> {
  const res = await fetch(proxyURL('/attestation/policy'), {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(policy),
  })
  if (!res.ok) throw new Error(`updateAttestationPolicy: ${res.status} ${res.statusText}`)
}
