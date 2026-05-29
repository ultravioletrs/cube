// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import path from 'path'
import type { IncomingMessage } from 'node:http'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

function bypassSPARoute(req: IncomingMessage): string | undefined {
  const accept = req.headers.accept ?? ''
  if (typeof accept === 'string' && accept.includes('text/html')) {
    return req.url
  }
  return undefined
}

export default defineConfig({
  plugins: [react(), tailwindcss()],

  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },

  server: {
    proxy: {
      '/users': {
        target: process.env['MG_USERS_PROXY_TARGET'] ?? 'https://localhost',
        changeOrigin: true,
        secure: false,
      },
      '/domains': {
        target: process.env['MG_DOMAINS_PROXY_TARGET'] ?? 'https://localhost',
        changeOrigin: true,
        secure: false,
        bypass: bypassSPARoute,
      },
      '/invitations': {
        target: process.env['MG_DOMAINS_PROXY_TARGET'] ?? 'https://localhost',
        changeOrigin: true,
        secure: false,
        bypass: bypassSPARoute,
      },
      '/journal': {
        target: process.env['MG_DOMAINS_PROXY_TARGET'] ?? 'https://localhost',
        changeOrigin: true,
        secure: false,
      },
      '/proxy': {
        target: process.env['MG_DOMAINS_PROXY_TARGET'] ?? 'https://localhost',
        changeOrigin: true,
        secure: false,
      },
      '/api/v1/chat': {
        target: process.env['EMBEDDER_PROXY_TARGET'] ?? 'http://localhost:8082',
        changeOrigin: true,
      },
      '/api': {
        target: process.env['EMBEDDER_PROXY_TARGET'] ?? 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
})
