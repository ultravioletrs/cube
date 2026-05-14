// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Feature flags — set via VITE_* environment variables at build time.
// All values default to disabled/empty so a vanilla deployment is unaffected.

export const ATTESTATION_ENABLED = import.meta.env.VITE_ENABLE_ATTESTATION === 'true'

// Base URL of the Cube Proxy service (e.g. http://localhost:9090).
// Leave empty in production when nginx routes /{domainID}/... to the proxy.
export const CUBE_PROXY_URL: string = import.meta.env.VITE_CUBE_PROXY_URL ?? ''
