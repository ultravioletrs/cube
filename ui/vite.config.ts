// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const cubeProxyTarget = process.env['CUBE_PROXY_TARGET'] ?? 'http://localhost:8900'

export default defineConfig({
  plugins: [react(), tailwindcss()],

  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },

  server: {
    proxy: {
      '/proxy': {
        target: cubeProxyTarget,
        changeOrigin: true,
        secure: false,
        rewrite: path => path.replace(/^\/proxy/, ''),
      },
      '^/[0-9a-fA-F-]{36}/': {
        target: cubeProxyTarget,
        changeOrigin: true,
        secure: false,
      },
      '/api': {
        target: cubeProxyTarget,
        changeOrigin: true,
        secure: false,
      },
    },
  },
})
