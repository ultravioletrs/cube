import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

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
        target: process.env['MG_USERS_PROXY_TARGET'] ?? 'http://localhost:9002',
        changeOrigin: true,
      },
      '/api/v1/chat': {
        target: process.env['CHAT_PROXY_TARGET'] ?? 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api': {
        target: process.env['EMBEDDER_PROXY_TARGET'] ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
