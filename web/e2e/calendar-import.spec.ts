import type { Locator, Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { stubCalendar, tomorrowEventTitle } from './helpers/calendar-stub'
import { e2eDataDir } from './helpers/paths'

const FIXTURE_ROOT = path.join(e2eDataDir, 'import-fixture')
const DROP_NAME = 'Library Drop'
const DROP_DIR = path.join(FIXTURE_ROOT, DROP_NAME)
const MOVIE_FOLDER = 'Oppenheimer (2023)'
const MOVIE_FILE = 'Oppenheimer.2023.1080p.BluRay.x264-GROUP.mkv'
const FILE_BYTES = 300 * 1024 * 1024
const FOLDER_COUNT = 24

function folderLabel(index: number): string {
  return `Folder ${index.toString().padStart(2, '0')}`
}

function createFixture(): void {
  for (let index = 1; index <= FOLDER_COUNT; index += 1) {
    fs.mkdirSync(path.join(DROP_DIR, folderLabel(index)), { recursive: true })
  }
  const movieDir = path.join(DROP_DIR, MOVIE_FOLDER)
  fs.mkdirSync(movieDir, { recursive: true })
  const file = path.join(movieDir, MOVIE_FILE)
  if (!fs.existsSync(file) || fs.statSync(file).size !== FILE_BYTES) {
    fs.writeFileSync(file, '')
    fs.truncateSync(file, FILE_BYTES)
  }
}

function importUrl(target: string): string {
  return `/import?path=${encodeURIComponent(target)}`
}

// The manual import pipeline cannot resolve a module entity for a match built
// by hand, so every real import answers with an error. The transport is stubbed
// so the row list's import action can be driven to completion.
async function stubManualImport(page: Page): Promise<void> {
  await page.route('**/api/v1/import/manual', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        sourcePath: '',
        destinationPath: '',
        isUpgrade: false,
        requiresSlotSelection: false,
        slotAssignments: [],
      }),
    })
  })
}

async function scrollRegion(region: Locator, top: number): Promise<void> {
  await region.evaluate((el, value) => {
    el.scrollTop = value
    el.dispatchEvent(new Event('scroll'))
  }, top)
}

async function regionScrollTop(region: Locator): Promise<number> {
  return region.evaluate((el) => el.scrollTop)
}

test.beforeAll(() => {
  createFixture()
})

test('calendar toggles between month and list', async ({ page, activate }) => {
  await stubCalendar(page)
  await page.goto('/calendar')
  await expect(page.getByRole('heading', { name: 'Calendar' })).toBeVisible()

  const toggle = page.getByRole('radiogroup', { name: 'Calendar view' })
  await expect(toggle.getByRole('radio', { name: 'Month' })).toHaveAttribute(
    'aria-checked',
    'true',
  )

  await activate(toggle.getByRole('radio', { name: 'List' }))
  await expect(toggle.getByRole('radio', { name: 'List' })).toHaveAttribute('aria-checked', 'true')
  await expect(page.getByRole('button', { name: /^Previous month$/ })).toBeHidden()
})

test('calendar list groups releases by day', async ({ page, activate }) => {
  await stubCalendar(page)
  await page.goto('/calendar')
  await activate(page.getByRole('radiogroup', { name: 'Calendar view' }).getByRole('radio', { name: 'List' }))

  const today = page.getByRole('region', { name: 'Calendar' }).locator('section', {
    has: page.getByRole('heading', { name: 'Today', exact: true }),
  })
  await expect(today.getByText('Stub Feature')).toBeVisible()
  await expect(today.getByText('Stub Series')).toBeVisible()
  await expect(today.getByText(tomorrowEventTitle)).toHaveCount(0)

  const tomorrow = page.getByRole('region', { name: 'Calendar' }).locator('section', {
    has: page.getByRole('heading', { name: 'Tomorrow', exact: true }),
  })
  await expect(tomorrow.getByText(tomorrowEventTitle)).toBeVisible()
})

test('calendar month view shows releases on their date', async ({ page, activate }, testInfo) => {
  await stubCalendar(page)
  await page.goto('/calendar')
  await expect(page.getByRole('heading', { name: 'Calendar' })).toBeVisible()

  const todayCell = page.getByRole('button', { name: /, 2 releases$/ })
  await expect(todayCell).toHaveCount(1)
  await activate(todayCell)
  await expect(page.getByText('Stub Feature')).toBeVisible()
  await expect(page.getByText('Stub Series')).toBeVisible()

  if (shellKind(testInfo.project.name) === 'phone') {
    const region = page.getByRole('region', { name: 'Calendar' })
    const overflow = await region.evaluate((el) => el.scrollWidth - el.clientWidth)
    expect(overflow).toBeLessThanOrEqual(1)
    const documentOverflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    )
    expect(documentOverflow).toBeLessThanOrEqual(1)
  }
})

test('manual import browser rows are tappable and folders show a chevron', async ({ page }) => {
  await page.goto(importUrl(DROP_DIR))
  const region = page.getByRole('region', { name: DROP_NAME })
  const folder = region.getByRole('button', { name: folderLabel(1) })
  await expect(folder).toBeVisible()
  const box = await folder.boundingBox()
  expect(box?.height ?? 0).toBeGreaterThanOrEqual(44)
  await expect(folder.locator('svg.lucide-chevron-right')).toHaveCount(1)

  await page.goto(importUrl(path.join(DROP_DIR, MOVIE_FOLDER)))
  const fileRow = page.getByRole('button', { name: MOVIE_FILE, exact: true })
  await expect(fileRow).toBeVisible()
  const fileBox = await fileRow.boundingBox()
  expect(fileBox?.height ?? 0).toBeGreaterThanOrEqual(44)
})

test('manual import shows the detected match and imports from the row', async ({ page, activate }) => {
  await stubManualImport(page)
  await page.goto(importUrl(path.join(DROP_DIR, MOVIE_FOLDER)))
  await expect(page.getByRole('heading', { name: MOVIE_FOLDER })).toBeVisible()

  const fileRow = page.getByRole('button', { name: MOVIE_FILE, exact: true })
  await expect(fileRow).toContainText('Oppenheimer (2023)')

  await activate(page.getByRole('button', { name: `Import ${MOVIE_FILE}` }))
  await expect(fileRow).toHaveCount(0)
})

test('entering a folder pushes a list that returns to the parent scroll position', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto(importUrl(DROP_DIR))
  const region = page.getByRole('region', { name: DROP_NAME })
  await expect(region.getByRole('button', { name: folderLabel(1) })).toBeVisible()

  await scrollRegion(region, 300)
  const scrolled = await regionScrollTop(region)
  expect(scrolled).toBeGreaterThan(0)

  await activate(region.getByRole('button', { name: folderLabel(20) }))
  await expect(page.getByRole('heading', { name: folderLabel(20) })).toBeVisible()
  await expect(page.getByRole('button', { name: DROP_NAME })).toBeVisible()

  await page.goBack()
  await expect(page.getByRole('heading', { name: DROP_NAME })).toBeVisible()
  await expect
    .poll(async () => Math.abs((await regionScrollTop(region)) - scrolled))
    .toBeLessThan(8)

  await activate(region.getByRole('button', { name: folderLabel(20) }))
  await expect(page.getByRole('heading', { name: folderLabel(20) })).toBeVisible()
  await activate(page.getByRole('button', { name: DROP_NAME }))
  await expect(page.getByRole('heading', { name: DROP_NAME })).toBeVisible()
})
