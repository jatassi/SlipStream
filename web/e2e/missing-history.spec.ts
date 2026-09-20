import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { apiGet, ensureDownloading } from './helpers/dev-data'

type MissingSeries = { id: number; title: string; missingCount: number }
type HistoryEntry = { id: number; mediaTitle?: string; eventType: string }

function moduleSegment(page: Page): Locator {
  return page.getByRole('radiogroup', { name: 'Missing module' })
}

function viewSegment(page: Page): Locator {
  return page.getByRole('radiogroup', { name: 'Missing view' })
}

async function firstMissingSeries(page: Page): Promise<MissingSeries> {
  const series = await apiGet<MissingSeries[]>(page, '/missing/series')
  if (!series[0]) {
    throw new Error('the developer library holds no series with missing episodes')
  }
  return series[0]
}

async function newestMovieHistory(page: Page): Promise<HistoryEntry> {
  const data = await apiGet<{ items: HistoryEntry[] }>(page, '/history?mediaType=movie&pageSize=20')
  const titled = data.items.find((item) => (item.mediaTitle ?? '') !== '')
  if (!titled?.mediaTitle) {
    throw new Error('history holds no titled movie entry')
  }
  return titled
}

async function openMissingSeries(page: Page, activate: (l: Locator) => Promise<void>): Promise<void> {
  await page.goto('/missing')
  await expect(page.getByRole('heading', { name: 'Missing', exact: true })).toBeVisible()
  await activate(moduleSegment(page).getByRole('radio', { name: 'Series' }))
}

test('missing segments the enabled modules and toggles missing or upgradable', async ({
  page,
  activate,
}) => {
  await page.goto('/missing')
  await expect(page.getByRole('heading', { name: 'Missing', exact: true })).toBeVisible()

  const modules = moduleSegment(page)
  await expect(modules.getByRole('radio', { name: 'Movies' })).toBeVisible()
  await expect(modules.getByRole('radio', { name: 'Series' })).toBeVisible()
  await expect(modules.getByRole('radio', { name: 'Movies' })).toHaveAttribute(
    'aria-checked',
    'true',
  )

  const series = await firstMissingSeries(page)
  await activate(modules.getByRole('radio', { name: 'Series' }))
  await expect(page.getByRole('link', { name: series.title, exact: true })).toBeVisible()
  await expect(page.getByText(/\d+ missing episodes?/).first()).toBeVisible()

  const views = viewSegment(page)
  await expect(views.getByRole('radio', { name: 'Missing' })).toHaveAttribute('aria-checked', 'true')
  await activate(views.getByRole('radio', { name: 'Upgradable' }))
  await expect(page.getByText(/\d+ missing episodes?/)).toHaveCount(0)
  await expect(
    page
      .getByText(/\d+ upgradable episodes?/)
      .or(page.getByRole('heading', { name: 'No upgradable episodes' }))
      .first(),
  ).toBeVisible()

  await activate(views.getByRole('radio', { name: 'Missing' }))
  await expect(page.getByText(/\d+ missing episodes?/).first()).toBeVisible()
})

test('a missing row opens the detail and searches in one tap', async ({ page, activate }) => {
  await openMissingSeries(page, activate)
  const series = await firstMissingSeries(page)
  await activate(page.getByRole('link', { name: series.title, exact: true }))
  await expect(page.getByRole('banner', { name: series.title })).toBeVisible()

  await openMissingSeries(page, activate)
  const actions = page.getByRole('group', { name: `Actions for ${series.title}` })
  await expect(actions.getByRole('button', { name: 'Auto Search' })).toBeVisible()
  await activate(actions.getByRole('button', { name: 'Auto Search' }))
  await expect(actions.getByText('Searching...')).toBeVisible()
})

test('history groups events by day and follows the filter', async ({ page, activate }) => {
  await page.goto('/')
  await ensureDownloading(page)
  const entry = await newestMovieHistory(page)
  const title = entry.mediaTitle ?? ''

  await page.goto('/history')
  await expect(page.getByRole('heading', { name: 'History', exact: true })).toBeVisible()

  const media = page.getByRole('radiogroup', { name: 'History media type' })
  await activate(media.getByRole('radio', { name: 'Movies' }))

  await expect(page.getByRole('heading', { name: 'Today' })).toBeVisible()
  const row = page.getByRole('link', { name: new RegExp(escapeForRegExp(title)) }).first()
  await expect(row).toBeVisible()
  await expect(row).toContainText(/Grabbed · |Imported · |Upgraded · /)
  await expect(row).toContainText(/Just now|ago/)

  await activate(media.getByRole('radio', { name: 'Series' }))
  await expect(page.getByRole('link', { name: new RegExp(escapeForRegExp(title)) })).toHaveCount(0)
})

test('phone missing and history read back to More', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/more')
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()

  await activate(page.getByRole('link', { name: 'Missing' }))
  await expect(page.getByRole('heading', { name: 'Missing', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'More' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'More' }))

  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await activate(page.getByRole('link', { name: 'History' }))
  await expect(page.getByRole('heading', { name: 'History', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'More' })).toBeVisible()
})

function escapeForRegExp(value: string): string {
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
}
