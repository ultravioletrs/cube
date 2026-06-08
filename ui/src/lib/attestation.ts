// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// AttestationPolicy is the raw JSON stored as JSONB on the proxy.
// The schema is intentionally open — it is defined by the attestation provider.
export type AttestationPolicy = Record<string, unknown>

function authHeaders(token: string): Record<string, string> {
  if (!token) return {}
  return { Authorization: `Bearer ${token}` }
}

// getAttestationPolicy fetches the latest attestation policy for a domain.
// Traefik routes /proxy/* → cube-proxy (strips the /proxy prefix).
// Endpoint: GET /proxy/{domainID}/attestation/policy
export async function getAttestationPolicy(
  token: string,
  domainID: string,
): Promise<AttestationPolicy | null> {
  const res = await fetch(`/proxy/${domainID}/attestation/policy`, {
    credentials: 'omit',
    headers: authHeaders(token),
  })
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`getAttestationPolicy: ${res.status} ${res.statusText}`)
  const body = await res.text()
  if (!body.trim()) return null
  return JSON.parse(body) as AttestationPolicy
}

// updateAttestationPolicy stores a new attestation policy.
// Endpoint: POST /proxy/attestation/policy
export async function updateAttestationPolicy(
  token: string,
  policy: AttestationPolicy,
): Promise<void> {
  const res = await fetch('/proxy/attestation/policy', {
    method: 'POST',
    credentials: 'omit',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(token),
    },
    body: JSON.stringify(policy),
  })
  if (!res.ok) throw new Error(`updateAttestationPolicy: ${res.status} ${res.statusText}`)
}
