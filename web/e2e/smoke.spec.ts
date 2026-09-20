import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'

const STATUS_LABEL = /^(Available|Missing|Downloading|Upgradable|Unreleased|Failed)$/

test('dashboard shows developer-mode data', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Health' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Storage' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Downloading' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Recent' })).toBeVisible()
})

test('library list shows items', async ({ page }) => {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
})

test('focusing search does not change visual viewport scale', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/search')
  const search = page.getByRole('searchbox', { name: 'Search' })
  await expect(search).toBeVisible()
  const before = await page.evaluate(readViewportScale)
  await activate(search)
  await search.focus()
  const after = await page.evaluate(readViewportScale)
  expect(after).toBe(before)
})

test('status marks have accessible text', async ({ page }) => {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  const marks = page.getByRole('img', { name: STATUS_LABEL })
  await expect(marks.first()).toBeVisible()
  expect(await marks.count()).toBeGreaterThan(0)
})

function readViewportScale(): number {
  return globalThis.visualViewport?.scale ?? 1
}
