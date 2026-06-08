// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import path from 'path'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    exclude: ['tests/smoke/**', '**/node_modules/**', '**/dist/**'],
    setupFiles: ['./src/test/setup.ts'],
    globals: false,
  },
})
