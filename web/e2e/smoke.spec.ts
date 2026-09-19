import { expect, test } from './fixtures'

const STATUS_LABEL = /^(Available|Missing|Downloading|Upgradable|Unreleased|Failed)$/

test('dashboard shows developer-mode data', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expect(page.getByText('System Health')).toBeVisible()
  await expect(page.getByText('Download Clients')).toBeVisible()
  await expect(page.getByText('Indexers')).toBeVisible()
  await expect(page.getByText('Active Downloads')).toBeVisible()
  await expect(page.getByText('Storage').first()).toBeVisible()
})

test('library list shows items', async ({ page }) => {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
})

test('focusing search does not change visual viewport scale', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'phone', 'phone profile only')
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  const search = page.getByPlaceholder('Search...')
  await expect(search).toBeVisible()
  const before = await page.evaluate(readViewportScale)
  await search.focus()
  const after = await page.evaluate(readViewportScale)
  expect(after).toBe(before)
})

test('status pills have accessible text', async ({ page }) => {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  const pills = page.getByText(STATUS_LABEL)
  await expect(pills.first()).toBeVisible()
  expect(await pills.count()).toBeGreaterThan(0)
})

function readViewportScale(): number {
  return globalThis.visualViewport?.scale ?? 1
}
