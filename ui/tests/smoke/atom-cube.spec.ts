// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
import { expect, test } from '@playwright/test'

const identifier = process.env.SMOKE_ATOM_IDENTIFIER
const password = process.env.SMOKE_ATOM_PASSWORD
const atomUIURL = process.env.ATOM_UI_URL ?? 'http://localhost:3005'
const atomAPIURL = process.env.ATOM_API_URL ?? 'http://localhost:8080'
const atomSessionKey = 'cube_atom_session'

test('unauthenticated users land on auth', async ({ page }) => {
  await page.goto('/dashboard')
  await expect(page).toHaveURL(/\/auth/)
  await expect(page.getByRole('heading', { name: /welcome back/i })).toBeVisible()
})

test.describe('authenticated Cube UI', () => {
  test.skip(!identifier || !password, 'SMOKE_ATOM_IDENTIFIER and SMOKE_ATOM_PASSWORD are required')

  test.beforeEach(async ({ page }) => {
    await page.goto('/auth')
    await page.getByLabel(/email or username/i).fill(identifier ?? '')
    await page.getByRole('textbox', { name: /^password$/i }).fill(password ?? '')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page).toHaveURL(/\/dashboard/)
  })

  test('loads protected Cube routes and ATOM workspace data', async ({ page }) => {
    const stored = await page.evaluate((key) => {
      const raw = window.localStorage.getItem(key)
      return raw ? JSON.parse(raw) as { accessToken?: string; entityId?: string; sessionId?: string; expiresAt?: string } : null
    }, atomSessionKey)
    expect(stored?.accessToken?.split('.')).toHaveLength(3)
    expect(stored?.entityId).toBeTruthy()
    expect(stored?.sessionId).toBeTruthy()
    expect(stored?.expiresAt).toBeTruthy()

    const atomCookies = await page.context().cookies(atomAPIURL)
    expect(atomCookies.some(cookie => cookie.name === 'atom_token')).toBe(false)

    await page.reload()
    await expect(page).toHaveURL(/\/dashboard/)
    await expect(page.getByText('Dashboard').first()).toBeVisible()

    for (const path of ['/records', '/sources', '/chat', '/config', '/guardrails']) {
      await page.goto(path)
      await expect(page.locator('body')).not.toContainText('Application error')
      await expect(page.locator('body')).not.toContainText('request failed')
    }

    await page.goto('/workspaces')
    await expect(page.getByRole('heading', { name: 'Workspaces', exact: true })).toBeVisible()

    const workspaceName = `cube-smoke-${Date.now()}`
    await page.getByRole('button', { name: /new workspace/i }).click()
    await page.getByLabel(/workspace name/i).fill(workspaceName)
    await page.getByRole('button', { name: /^create$/i }).click()
    const workspaceEntry = page.getByText(workspaceName).first()
    await expect(workspaceEntry).toBeVisible()
    const recordsLoadAfterSelect = page.waitForResponse(response => (
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/records')
    ))
    const sourcesLoadAfterSelect = page.waitForResponse(response => (
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/sources')
    ))
    await workspaceEntry.click()
    expect((await recordsLoadAfterSelect).status()).toBe(200)
    expect((await sourcesLoadAfterSelect).status()).toBe(200)

    await page.goto('/records')
    await expect(page.getByRole('heading', { name: 'Records', exact: true })).toBeVisible()
    await expect(page.locator('body')).not.toContainText('request failed')

    await page.goto('/sources')
    await expect(page.getByRole('heading', { name: 'Sources', exact: true })).toBeVisible()
    await expect(page.locator('body')).not.toContainText('request failed')

    await page.goto('/config')
    await expect(page.locator('body')).toContainText('llama3.2:3b', { timeout: 30_000 })
    await expect(page.locator('body')).not.toContainText('No local models found')

    await page.goto('/chat')
    await page.getByPlaceholder(/ask a question/i).fill('Say hello in one short sentence.')
    const chatResponse = page.waitForResponse(response => (
      response.request().method() === 'POST' &&
      response.url().includes('/api/v1/chat')
    ))
    await page.getByRole('button', { name: /^send$/i }).click()
    await expect(page.locator('body')).not.toContainText('Cannot answer yet')
    expect((await chatResponse).status()).toBe(200)

    await page.goto('/dashboard')
    await expect(page.getByText('Conversations Today')).toBeVisible()
    await expect(page.locator('body')).not.toContainText('OpenSearch error')

    await page.goto('/attestation')
    await expect(page.locator('body')).not.toContainText('Application error')

    await page.goto('/members')
    await expect(page.getByRole('heading', { name: 'Members', exact: true })).toBeVisible()

    await page.goto('/invitations')
    await expect(page.getByRole('heading', { name: 'Invitations', exact: true })).toBeVisible()

    await page.goto('/audit-logs')
    await expect(page.getByRole('heading', { name: 'Audit Logs', exact: true })).toBeVisible()

    await page.evaluate(() => {
      window.open = ((url?: string | URL | undefined) => {
        ;(window as typeof window & { __cubeOpenedURL?: string }).__cubeOpenedURL = String(url ?? '')
        return null
      }) as typeof window.open
    })
    await page.getByRole('button', { name: /advanced iam/i }).click()
    await expect.poll(() => (
      page.evaluate(() => (window as typeof window & { __cubeOpenedURL?: string }).__cubeOpenedURL)
    )).toBe(atomUIURL)

    await page.getByRole('button').filter({ hasText: /^[A-Z0-9]{1,2}$/ }).first().click()
    await page.getByRole('button', { name: /sign out/i }).click()
    await expect.poll(() => page.evaluate((key) => window.localStorage.getItem(key), atomSessionKey)).toBeNull()
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/auth/)
  })
})
