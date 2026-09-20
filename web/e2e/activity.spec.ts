import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { type ShellKind, shellKind } from './helpers/activate'
import {
  addOfflineDownloadClient,
  ensureSeriesDownloading,
  removeDownloadClient,
} from './helpers/dev-data'
import { installQueueStub, type StubQueueItem } from './helpers/queue-stub'

// These tests share the developer-mode queue with each other and with the
// dashboard spec, so they run one at a time and hold the queue to three items:
// any more and the dashboard's three-row Downloading group turns racy.
test.describe.configure({ mode: 'serial' })

// Every real grab here is an episode of a series of the project's own, so the
// two projects stay out of each other's way and out of the dashboard spec's.
// Both series are seeded with whole seasons still missing, so they keep
// episodes to grab; movies run out once the mock client has finished
// importing them, and Breaking Bad and Game of Thrones are seeded complete.
const SERIES_TITLE: Record<ShellKind, string> = {
  phone: 'The Boys',
  wide: 'The Mandalorian',
}

const STUB_DOWNLOADING: StubQueueItem = {
  id: 'stub-downloading',
  title: 'Stub Downloading',
  releaseName: 'Stub.Downloading.2019.1080p.BluRay.x264-STUB',
  mediaType: 'movie',
  status: 'downloading',
  progress: 32.5,
}

const STUB_QUEUED: StubQueueItem = {
  id: 'stub-queued',
  title: 'Stub Queued',
  releaseName: 'Stub.Queued.2021.1080p.WEB-DL-STUB',
  mediaType: 'movie',
  status: 'queued',
  progress: 0,
}

const STUB_IMPORTING: StubQueueItem = {
  id: 'stub-importing',
  title: 'Stub Importing',
  releaseName: 'Stub.Importing.2022.2160p.BluRay-STUB',
  mediaType: 'series',
  status: 'completed',
  progress: 100,
  season: 1,
  episode: 2,
}

test('queue rows carry thumbnail, release name, progress line and tabular numbers', async ({ page }, testInfo) => {
  await page.goto('/downloads')
  await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
  const download = await ensureSeriesDownloading(page, SERIES_TITLE[shellKind(testInfo.project.name)])
  await page.reload()

  await expect(page.getByText(/^\d+ downloading · .+ · \d+ in queue$/)).toBeVisible()
  const row = queueRow(page, download.releaseName)
  await expect(row).toBeVisible()
  await expect(row.getByRole('img', { name: download.rowTitle })).toBeVisible()
  await expect(row.getByText(download.rowTitle, { exact: true })).toBeVisible()
  await expect(row.getByText(download.releaseName)).toBeVisible()
  await expect(row.getByRole('progressbar')).toBeVisible()
  const stats = row.getByText(/%\s·/)
  await expect(stats).toBeVisible()
  expect(await numeralStyle(stats)).toContain('tabular-nums')
})

test('the trailing control pauses and resumes without opening anything', async ({ page, activate }, testInfo) => {
  await page.goto('/downloads')
  const download = await ensureSeriesDownloading(page, SERIES_TITLE[shellKind(testInfo.project.name)])
  await page.reload()
  const row = queueRow(page, download.releaseName)
  await expect(row).toBeVisible()

  await activate(page.getByRole('button', { name: `Pause ${download.rowTitle}` }))
  await expect(row.getByText(/· Paused ·/)).toBeVisible()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('menu')).toHaveCount(0)

  await activate(page.getByRole('button', { name: `Resume ${download.rowTitle}` }))
  await expect(row.getByText(/left$/)).toBeVisible()
})

test('queued and importing items show their state where the control would be', async ({ page }) => {
  await installQueueStub(page, [STUB_DOWNLOADING, STUB_QUEUED, STUB_IMPORTING])
  await page.goto('/downloads')

  const queued = queueRow(page, STUB_QUEUED.releaseName)
  const importing = queueRow(page, STUB_IMPORTING.releaseName)
  await expect(queued.getByText(/^Queued · /)).toBeVisible()
  await expect(importing.getByText(/^Importing · /)).toBeVisible()
  await expect(page.getByRole('button', { name: new RegExp(`^(Pause|Resume) ${STUB_QUEUED.title}$`) })).toHaveCount(0)
  await expect(page.getByRole('button', { name: new RegExp(`^(Pause|Resume) ${STUB_IMPORTING.title}$`) })).toHaveCount(0)
  await expect(page.getByRole('button', { name: `Pause ${STUB_DOWNLOADING.title}` })).toBeVisible()
})

test('the segmented filter narrows the queue to movies, series and back to all', async ({ page, activate }) => {
  await installQueueStub(page, [STUB_DOWNLOADING, STUB_IMPORTING])
  await page.goto('/downloads')
  const movie = queueRow(page, STUB_DOWNLOADING.releaseName)
  const series = queueRow(page, STUB_IMPORTING.releaseName)
  const filter = page.getByRole('radiogroup', { name: 'Filter downloads' })
  await expect(movie).toBeVisible()
  await expect(series).toBeVisible()

  await activate(filter.getByRole('radio', { name: 'Movies' }))
  await expect(movie).toBeVisible()
  await expect(series).toHaveCount(0)

  await activate(filter.getByRole('radio', { name: 'Series' }))
  await expect(series).toBeVisible()
  await expect(movie).toHaveCount(0)

  await activate(filter.getByRole('radio', { name: 'All' }))
  await expect(movie).toBeVisible()
  await expect(series).toBeVisible()

  await filter.getByRole('radio', { name: 'All' }).focus()
  await page.keyboard.press('ArrowRight')
  await expect(filter.getByRole('radio', { name: 'Movies' })).toHaveAttribute('aria-checked', 'true')
  await expect(series).toHaveCount(0)
})

test('an unreachable download client shows an amber row above the queue', async ({ page }, testInfo) => {
  await page.goto('/downloads')
  const download = await ensureSeriesDownloading(page, SERIES_TITLE[shellKind(testInfo.project.name)])
  const clientName = `Unreachable ${testInfo.project.name}`
  const clientId = await addOfflineDownloadClient(page, clientName)
  const warning = page.getByText(`Unable to reach ${clientName}`)

  try {
    await page.reload()
    await expect(warning).toBeVisible()
    const rowTitle = queueRow(page, download.releaseName).getByText(download.rowTitle, { exact: true })
    expect(await colorOf(warning)).not.toEqual(await colorOf(rowTitle))
    const warningBox = await warning.boundingBox()
    const firstRow = await page.getByRole('progressbar').first().boundingBox()
    expect(warningBox).not.toBeNull()
    expect(firstRow).not.toBeNull()
    expect(warningBox?.y ?? 0).toBeLessThan(firstRow?.y ?? 0)
  } finally {
    await removeDownloadClient(page, clientId)
  }

  await page.reload()
  await expect(warning).toHaveCount(0)
})

test('progress between two updates renders intermediate widths', async ({ page }) => {
  const stub = await installQueueStub(page, [{ ...STUB_DOWNLOADING, progress: 8 }])
  await page.goto('/downloads')
  const bar = queueRow(page, STUB_DOWNLOADING.releaseName).getByRole('progressbar')
  await expect(bar).toHaveAttribute('aria-valuenow', '8')

  const sampling = sampleFillRatios(bar)
  stub.push([{ ...STUB_DOWNLOADING, progress: 88 }])
  const ratios = await sampling

  await expect(bar).toHaveAttribute('aria-valuenow', '88')
  expect(ratios.some((ratio) => ratio > 0.15 && ratio < 0.8)).toBe(true)
  expect(ratios.at(-1) ?? 0).toBeGreaterThan(0.8)
})

// The removals run last: each project grabs its series once per run, the
// filter check reuses it, and only then is it taken out of the queue.
test('phone row tap opens the action sheet and removing takes the row away', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/downloads')
  const download = await ensureSeriesDownloading(page, SERIES_TITLE.phone)
  await page.reload()

  await activate(queueRow(page, download.releaseName))
  const sheet = page.getByRole('dialog', { name: download.rowTitle })
  await expect(sheet).toBeVisible()
  const pause = sheet.getByRole('button', { name: 'Pause' })
  const remove = sheet.getByRole('button', { name: 'Remove from queue' })
  await expect(pause).toBeVisible()
  await expect(remove).toBeVisible()
  await expect(sheet.getByRole('button', { name: 'Cancel' })).toBeVisible()
  expect(await colorOf(remove)).not.toEqual(await colorOf(pause))
  expect(await gapBetween(pause, remove)).toBeGreaterThan(4)

  await activate(remove)
  const confirm = page.getByRole('dialog', { name: 'Remove from queue' })
  await expect(confirm.getByText(download.rowTitle)).toBeVisible()
  await activate(confirm.getByRole('button', { name: 'Remove', exact: true }))
  await expect(queueRow(page, download.releaseName)).toHaveCount(0)
})

test('wide row action opens a menu and removing confirms in a dialog', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/downloads')
  const download = await ensureSeriesDownloading(page, SERIES_TITLE.wide)
  await page.reload()

  await activate(page.getByRole('button', { name: `More actions for ${download.rowTitle}` }))
  const menu = page.getByRole('menu')
  await expect(menu.getByRole('menuitem', { name: 'Pause' })).toBeVisible()
  const remove = menu.getByRole('menuitem', { name: 'Remove from queue' })
  await expect(remove).toBeVisible()
  await activate(remove)

  const dialog = page.getByRole('alertdialog', { name: 'Remove from queue' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText(download.rowTitle)).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible()
  await activate(dialog.getByRole('button', { name: 'Remove', exact: true }))
  await expect(queueRow(page, download.releaseName)).toHaveCount(0)
})

function escapeRegExp(value: string): string {
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
}

function queueRow(page: Page, releaseName: string): Locator {
  return page.getByRole('button', { name: new RegExp(escapeRegExp(releaseName)) })
}

async function numeralStyle(locator: Locator): Promise<string> {
  return locator.evaluate((el) => globalThis.getComputedStyle(el).fontVariantNumeric)
}

async function colorOf(locator: Locator): Promise<string> {
  return locator.evaluate((el) => globalThis.getComputedStyle(el).color)
}

async function gapBetween(first: Locator, second: Locator): Promise<number> {
  const top = await first.boundingBox()
  const bottom = await second.boundingBox()
  if (top === null || bottom === null) {
    return 0
  }
  return bottom.y - (top.y + top.height)
}

async function sampleFillRatios(bar: Locator): Promise<number[]> {
  return bar.evaluate(
    (el) =>
      new Promise<number[]>((resolve) => {
        const ratios: number[] = []
        const started = performance.now()
        const tick = () => {
          const fill = el.firstElementChild
          const track = el.getBoundingClientRect().width
          if (fill !== null && track > 0) {
            ratios.push(fill.getBoundingClientRect().width / track)
          }
          if (performance.now() - started < 900) {
            requestAnimationFrame(tick)
            return
          }
          resolve(ratios)
        }
        tick()
      }),
  )
}
