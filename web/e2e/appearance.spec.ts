import type { Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { findMovie } from './helpers/dev-data'
import { primaryNav, sidebarNav } from './helpers/shell'
import { readThemeIsDark, useTheme } from './helpers/theme'

const MOVIE_TITLE = 'The Matrix'

const STATUSES = [
  'available',
  'missing',
  'downloading',
  'upgradable',
  'unreleased',
  'failed',
] as const

const STATUS_LABEL: Record<string, (typeof STATUSES)[number] | undefined> = {
  Available: 'available',
  Missing: 'missing',
  Downloading: 'downloading',
  Upgradable: 'upgradable',
  Unreleased: 'unreleased',
  Failed: 'failed',
}

/**
 * Every status mark's painted colour, keyed by its accessible name, next to the
 * palette the theme resolves for the same status. Both go through the browser's
 * own colour serialisation, so the two are comparable.
 */
async function readStatusColours(page: Page): Promise<{ marks: [string, string][]; palette: Record<string, string> }> {
  return page.evaluate((statuses) => {
    const probe = document.createElement('span')
    document.body.append(probe)
    const palette: Record<string, string> = {}
    for (const status of statuses) {
      probe.style.backgroundColor = `var(--status-${status})`
      palette[status] = getComputedStyle(probe).backgroundColor
    }
    probe.remove()

    const marks: [string, string][] = []
    document.querySelectorAll('[role="img"][aria-label]').forEach((element) => {
      marks.push([
        element.getAttribute('aria-label') ?? '',
        getComputedStyle(element).backgroundColor,
      ])
    })
    return { marks, palette }
  }, STATUSES)
}

type StatusMark = { status: (typeof STATUSES)[number]; colour: string }

function statusMarks(marks: [string, string][]): StatusMark[] {
  const found: StatusMark[] = []
  for (const [label, colour] of marks) {
    const status = STATUS_LABEL[label]
    if (status !== undefined) {
      found.push({ status, colour })
    }
  }
  return found
}

async function openLibrary(page: Page): Promise<void> {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: new RegExp(MOVIE_TITLE) })).toBeVisible()
}

async function openDetail(page: Page): Promise<void> {
  await page.goto('/movies')
  const movie = await findMovie(page, MOVIE_TITLE)
  await page.goto(`/movies/${movie.id}`)
  await expect(page.getByRole('heading', { name: movie.title, level: 1 })).toBeVisible()
}

test('the light theme paints status marks from the light palette', async ({ page }) => {
  await useTheme(page, 'light')
  await openLibrary(page)
  expect(await readThemeIsDark(page)).toBe(false)

  const library = await readStatusColours(page)
  const found = statusMarks(library.marks)
  expect(found.length).toBeGreaterThan(0)
  for (const mark of found) {
    expect(mark.colour, `${mark.status} mark`).toBe(library.palette[mark.status])
  }

  await openDetail(page)
  const detail = await readStatusColours(page)
  for (const mark of statusMarks(detail.marks)) {
    expect(mark.colour, `${mark.status} mark on the detail`).toBe(detail.palette[mark.status])
  }
})

test('the light and dark palettes are distinct', async ({ page }) => {
  await useTheme(page, 'dark')
  await openLibrary(page)
  expect(await readThemeIsDark(page)).toBe(true)
  const dark = await readStatusColours(page)

  await useTheme(page, 'light')
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  expect(await readThemeIsDark(page)).toBe(false)
  const light = await readStatusColours(page)

  for (const status of STATUSES) {
    expect(light.palette[status], status).not.toBe(dark.palette[status])
  }
})

test('reduced transparency renders the bars solid', async ({ page }) => {
  await openLibrary(page)
  const translucent = await readBarMaterials(page)
  expect(translucent.length).toBeGreaterThan(0)
  expect(translucent.some((bar) => bar.backdropFilter !== 'none')).toBe(true)

  const client = await page.context().newCDPSession(page)
  await client.send('Emulation.setEmulatedMedia', {
    features: [{ name: 'prefers-reduced-transparency', value: 'reduce' }],
  })

  for (const open of [openLibrary, openDetail, openDashboard, openSettings]) {
    await open(page)
    const bars = await readBarMaterials(page)
    expect(bars.length, 'material bars on screen').toBeGreaterThan(0)
    for (const bar of bars) {
      expect(bar.backdropFilter).toBe('none')
      expect(bar.background).toBe(bar.pageBackground)
    }
  }
})

test('the list entrance plays once and is not replayed on return', async ({
  page,
  activate,
}, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await recordEntrances(page)

  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expect.poll(() => countEntrances(page), { timeout: 10_000 }).toBeGreaterThan(0)
  // The groups are staggered, so their entrances start over the next 200 ms.
  await page.waitForTimeout(600)
  await page.evaluate(() => {
    document.documentElement.dataset.entrances = '0'
  })

  const nav = kind === 'phone' ? primaryNav(page) : sidebarNav(page)
  await activate(nav.getByRole('link', { name: kind === 'phone' ? 'Library' : 'Movies' }))
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()

  await page.goBack()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()

  for (let sample = 0; sample < 12; sample += 1) {
    expect(await countEntrances(page), `sample ${sample}`).toBe(0)
    await page.waitForTimeout(50)
  }
})

async function openDashboard(page: Page): Promise<void> {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
}

async function openSettings(page: Page): Promise<void> {
  await page.goto('/settings/media/quality-profiles')
  await expect(page.getByRole('heading', { name: 'Quality Profiles' })).toBeVisible()
}

type BarMaterial = {
  backdropFilter: string
  background: string
  pageBackground: string
}

async function readBarMaterials(page: Page): Promise<BarMaterial[]> {
  return page.evaluate(() => {
    type BarMaterial = { backdropFilter: string; background: string; pageBackground: string }

    const probe = document.createElement('span')
    probe.style.backgroundColor = 'var(--background)'
    document.body.append(probe)
    const pageBackground = getComputedStyle(probe).backgroundColor
    probe.remove()

    const bars: BarMaterial[] = []
    document.querySelectorAll('.material, .material-heavy, .material-chip').forEach((element) => {
      const style = getComputedStyle(element)
      bars.push({
        backdropFilter: style.backdropFilter,
        background: style.backgroundColor,
        pageBackground,
      })
    })
    return bars
  })
}

/**
 * Entrance animations are short enough to be over before a locator resolves, so
 * they are counted as they start rather than caught mid-flight.
 */
async function recordEntrances(page: Page): Promise<void> {
  await page.addInitScript(() => {
    document.addEventListener(
      'animationstart',
      (event) => {
        if (event.animationName.startsWith('enter-')) {
          const root = document.documentElement
          root.dataset.entrances = String(Number(root.dataset.entrances ?? 0) + 1)
        }
      },
      true,
    )
  })
}

async function countEntrances(page: Page): Promise<number> {
  return page.evaluate(() => Number(document.documentElement.dataset.entrances ?? 0))
}
