import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { type DownloadRef, ensureDownloading, ensureHealthIssue, ensureRecentHistory } from './helpers/dev-data'
import { primaryNav } from './helpers/shell'

test('dashboard shows grouped Health Storage Downloading and Recent', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Health' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Storage' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Downloading' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Recent' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Storage' }).getByText(/ used$/)).toBeVisible()
  await expect(page.getByText('Mock Movies')).toBeVisible()
  await expect(page.getByText('Mock TV')).toBeVisible()
})

test('health issue row uses a warning tile and opens System', async ({ page, activate }) => {
  await page.goto('/')
  const issue = await ensureHealthIssue(page)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  const row = page.getByRole('region', { name: 'Health' }).getByRole('link', { name: new RegExp(escapeRegExp(issue.name)) })
  await expect(row).toBeVisible()
  await expect(row.getByText(issue.message ?? issue.name)).toBeVisible()
  await activate(row)
  await expect(page.getByRole('heading', { name: 'System' })).toBeVisible()
})

test('storage shows used total and a line per root folder', async ({ page }) => {
  await page.goto('/')
  const storage = page.getByRole('region', { name: 'Storage' })
  await expect(storage.getByText(/ used$/)).toBeVisible()
  await expect(storage.getByText(/^of /)).toBeVisible()
  await expect(storage.getByText(/Movies Mock Movies/)).toBeVisible()
  await expect(storage.getByText(/Series Mock TV/)).toBeVisible()
})

test('downloading lists at most three items and see all opens Activity', async ({ page, activate }) => {
  await page.goto('/')
  const download = await ensureDownloading(page)
  await page.reload()
  const downloading = page.getByRole('region', { name: 'Downloading' })
  await expect(downloading.getByRole('link', { name: 'See all' })).toBeVisible()
  await expect(downloadRow(downloading, download)).toBeVisible()
  const bars = downloading.getByRole('progressbar')
  await expect(bars.first()).toBeVisible()
  expect(await bars.count()).toBeLessThanOrEqual(3)
  await expect(downloading.getByText(/% · /).first()).toBeVisible()
  await activate(downloading.getByRole('link', { name: 'See all' }))
  await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
})

test('activating a downloading row opens that title', async ({ page, activate }) => {
  await page.goto('/')
  const download = await ensureDownloading(page)
  await page.reload()
  await activate(downloadRow(page.getByRole('region', { name: 'Downloading' }), download))
  await expectTitleDetail(page, download.detailTitle)
})

test('recent rows show event title and relative time and open the title', async ({ page, activate }) => {
  await page.goto('/')
  const entry = await ensureRecentHistory(page)
  await page.reload()
  const title = entry.mediaTitle ?? ''
  expect(title.length).toBeGreaterThan(0)
  const recent = page.getByRole('region', { name: 'Recent' })
  const row = recent.getByRole('link', { name: new RegExp(escapeRegExp(title)) }).first()
  await expect(row).toBeVisible()
  await expect(row.getByText(/Grabbed|Imported|Upgraded|Failed|Auto/)).toBeVisible()
  await expect(row.getByText(/ago$|Just now/)).toBeVisible()
  await activate(row)
  await expectTitleDetail(page, title)
})

test('groups stack on phone and sit in two columns on wide', async ({ page }, testInfo) => {
  await page.goto('/')
  const health = page.getByRole('region', { name: 'Health' })
  const storage = page.getByRole('region', { name: 'Storage' })
  await expect(health).toBeVisible()
  await expect(storage).toBeVisible()
  const healthBox = await health.boundingBox()
  const storageBox = await storage.boundingBox()
  expect(healthBox).not.toBeNull()
  expect(storageBox).not.toBeNull()
  if (healthBox === null || storageBox === null) {
    return
  }
  if (shellKind(testInfo.project.name) === 'phone') {
    expect(storageBox.y).toBeGreaterThan(healthBox.y + healthBox.height - 1)
    expect(Math.abs(storageBox.x - healthBox.x)).toBeLessThan(8)
    expect(storageBox.y - (healthBox.y + healthBox.height)).toBeGreaterThanOrEqual(16)
    return
  }
  expect(storageBox.x).toBeGreaterThan(healthBox.x + healthBox.width)
  expect(Math.abs(storageBox.y - healthBox.y)).toBeLessThan(8)
})

test('skeleton rows match loaded row height', async ({ page, activate }, testInfo) => {
  await page.goto('/')
  const health = page.getByRole('region', { name: 'Health' })
  const loaded = health.getByRole('link').first()
  await expect(loaded).toBeVisible()
  const loadedBox = await loaded.boundingBox()
  expect(loadedBox).not.toBeNull()
  await setForceLoading({ page, activate, kind: shellKind(testInfo.project.name), on: true })
  const skeleton = health.getByRole('status', { name: 'Loading' }).first()
  await expect(skeleton).toBeVisible()
  const skeletonBox = await skeleton.boundingBox()
  expect(skeletonBox).not.toBeNull()
  if (loadedBox === null || skeletonBox === null) {
    return
  }
  expect(Math.abs(skeletonBox.height - loadedBox.height)).toBeLessThanOrEqual(4)
  await expect(health.getByRole('status', { name: 'Loading' }).first()).toBeVisible()
})

function downloadRow(downloading: Locator, download: DownloadRef): Locator {
  return downloading.getByRole('link', { name: new RegExp(escapeRegExp(download.rowTitle)) })
}

function escapeRegExp(value: string): string {
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
}

async function expectTitleDetail(page: Page, title: string): Promise<void> {
  await expect(page).toHaveURL(/\/(movies|series)\/\d+/)
  await expect(
    page.getByRole('heading', { name: title }).or(page.getByRole('img', { name: title })),
  ).toBeVisible()
}

type ForceLoadingArgs = {
  page: Page
  activate: (locator: Locator) => Promise<void>
  kind: 'phone' | 'wide'
  on: boolean
}

async function setForceLoading({ page, activate, kind, on }: ForceLoadingArgs): Promise<void> {
  await activate(
    kind === 'phone'
      ? primaryNav(page).getByRole('link', { name: 'More' })
      : page.getByRole('button', { name: 'Developer Tools' }),
  )
  const label = page.getByText('Force Loading', { exact: true })
  await expect(label).toBeVisible()
  const toggle = page.getByRole('switch', { name: 'Force Loading' })
  if ((await toggle.isChecked()) !== on) {
    await activate(label)
    await expect(toggle).toHaveAttribute('aria-checked', on ? 'true' : 'false')
  }
  if (kind === 'phone') {
    await activate(primaryNav(page).getByRole('link', { name: 'Dashboard' }))
  }
}
