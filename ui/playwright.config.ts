// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/smoke',
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    baseURL: process.env.CUBE_UI_URL ?? 'http://localhost:6193',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
