import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { apiGet } from './helpers/dev-data'

const STATUS_LABEL = /^(Available|Missing|Downloading|Upgradable|Unreleased|Failed)$/

function screen(page: Page): Locator {
  return page.getByRole('region', { name: 'Search', exact: true })
}

function field(page: Page): Locator {
  return screen(page).getByRole('searchbox', { name: 'Search' })
}

function addNew(page: Page): Locator {
  return page.getByRole('region', { name: 'Add new', exact: true })
}

function fontSize(locator: Locator): Promise<number> {
  return locator.evaluate((element) => Number.parseFloat(globalThis.getComputedStyle(element).fontSize))
}

function readViewportScale(): number {
  return globalThis.visualViewport?.scale ?? 1
}

async function searchableCount(page: Page): Promise<number> {
  const movies = await apiGet<unknown[]>(page, '/movies')
  const series = await apiGet<unknown[]>(page, '/series')
  return movies.length + series.length
}

test('the search field never zooms the page and clears back to the empty state', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/search')
  const input = field(page)
  await expect(input).toBeVisible()
  expect(await fontSize(input)).toBeGreaterThanOrEqual(16)

  const before = await page.evaluate(readViewportScale)
  await activate(input)
  await input.focus()
  expect(await page.evaluate(readViewportScale)).toBe(before)

  await input.fill('The Matrix')
  await expect(screen(page).getByRole('link', { name: /The Matrix/ })).toBeVisible()

  await activate(page.getByRole('button', { name: 'Clear search' }))
  await expect(input).toHaveValue('')
  await expect(screen(page).getByText(/Search \d+ titles across movies and series\./)).toBeVisible()
})

test('the empty state names the number of searchable titles', async ({ page }) => {
  await page.goto('/search')
  const count = await searchableCount(page)
  expect(count).toBeGreaterThan(0)
  await expect(
    screen(page).getByText(`Search ${count} titles across movies and series.`),
  ).toBeVisible()
})

test('a library result row carries status, year and media type and opens the detail', async ({
  page,
  activate,
}) => {
  await page.goto('/search')
  await field(page).fill('The Matrix')
  const row = screen(page).getByRole('link', { name: /The Matrix/ }).first()
  await expect(row).toBeVisible()
  await expect(row).toHaveAttribute('href', /\/movies\/\d+$/)
  await expect(row.getByRole('img', { name: STATUS_LABEL })).toBeVisible()
  await expect(row.getByText('1999')).toBeVisible()
  await expect(row.getByText('Movie', { exact: true })).toBeVisible()

  await activate(row)
  await expect(page).toHaveURL(/\/movies\/\d+$/)
  await expect(page.getByRole('banner', { name: 'The Matrix' })).toBeVisible()
})

test('the add new entry opens the external search and continues into the add flow', async ({
  page,
  activate,
}) => {
  await page.goto('/search')
  await field(page).fill('the')
  await expect(screen(page).getByRole('link', { name: /The Matrix/ })).toBeVisible()

  await activate(page.getByRole('button', { name: /Add new/ }))
  await expect(addNew(page)).toBeVisible()

  const external = addNew(page).getByRole('button', { name: /The Dark Knight/ })
  await expect(external).toBeVisible()
  await activate(external)
  await expect(page).toHaveURL(/\/movies\/add\?tmdbId=155$/)
})

test('the wide header field searches from the dashboard', async ({ page }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  const header = page.getByRole('banner', { name: 'App' }).getByRole('searchbox', { name: 'Search' })
  await expect(header).toBeVisible()
  expect(await fontSize(header)).toBeGreaterThanOrEqual(16)

  await header.fill('The Matrix')
  await header.press('Enter')
  await expect(page).toHaveURL(/\/search\?q=/)
  const row = screen(page).getByRole('link', { name: /The Matrix/ }).first()
  await expect(row).toBeVisible()
  await expect(row).toHaveAttribute('href', /\/movies\/\d+$/)
})
