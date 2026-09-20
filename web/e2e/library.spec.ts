import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { apiGet } from './helpers/dev-data'

const STATUS_LABEL = /^(Available|Missing|Downloading|Upgradable|Unreleased|Failed)$/

type LibraryMovie = { id: number; title: string; status: string }

function libraryRegion(page: Page, title: string): Locator {
  return page.getByRole('region', { name: title })
}

function cells(page: Page, title: string): Locator {
  return page.getByRole('list', { name: title, exact: true }).getByRole('link')
}

function cell(page: Page, title: string, name: string): Locator {
  return cells(page, title).filter({ has: page.getByText(name, { exact: true }) })
}

function moduleSwitch(page: Page): Locator {
  return page.getByRole('radiogroup', { name: 'Library' })
}

function chip(page: Page, name: string): Locator {
  return page.getByRole('button', { name, exact: true })
}

// The number of cells sharing the first row's top edge is the grid's column count.
async function columnCount(grid: Locator): Promise<number> {
  return grid.evaluateAll((elements) => {
    const tops = elements.map((element) => Math.round(element.getBoundingClientRect().top))
    return tops.filter((top) => top === tops[0]).length
  })
}

async function openOptions(page: Page, activate: (locator: Locator) => Promise<void>): Promise<void> {
  await activate(page.getByRole('button', { name: 'Options', exact: true }))
}

test('library switches module and remembers the choice', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(moduleSwitch(page).getByRole('radio', { name: 'Movies' })).toHaveAttribute(
    'aria-checked',
    'true',
  )

  await activate(moduleSwitch(page).getByRole('radio', { name: 'Series' }))
  await expect(page.getByRole('heading', { name: 'Series' })).toBeVisible()
  await expect(page).toHaveURL(/\/series$/)
  await expect(cells(page, 'Series').first()).toBeVisible()

  await page.reload()
  await expect(page.getByRole('heading', { name: 'Series' })).toBeVisible()
  await expect(moduleSwitch(page).getByRole('radio', { name: 'Series' })).toHaveAttribute(
    'aria-checked',
    'true',
  )

  if (kind === 'phone') {
    const nav = page.getByRole('navigation', { name: 'Primary' })
    await activate(nav.getByRole('link', { name: 'Dashboard' }))
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
    await activate(nav.getByRole('link', { name: 'Library' }))
    await expect(page.getByRole('heading', { name: 'Series' })).toBeVisible()
  }
})

test('phone library is a three-column poster grid with no table view', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  const grid = cells(page, 'Movies')
  await expect(grid.first()).toBeVisible()
  expect(await columnCount(grid)).toBe(3)

  const first = grid.first()
  await expect(first).toContainText(/(?:19|20)\d{2}/)
  await expect(first).toContainText(/·\s*\S/)
  await expect(first.getByRole('img', { name: STATUS_LABEL })).toBeVisible()

  await openOptions(page, activate)
  await expect(page.getByText('Sort by Title')).toBeVisible()
  await expect(page.getByText(/^Table view/)).toHaveCount(0)
  await expect(page.getByText(/^Columns$/)).toHaveCount(0)
})

test('wide grid follows the poster size and keeps the table view', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/movies')
  const grid = cells(page, 'Movies')
  await expect(grid.first()).toBeVisible()

  await openOptions(page, activate)
  await activate(page.getByRole('menuitem', { name: 'Poster size' }))
  await activate(page.getByRole('menuitem', { name: /^Small/ }))
  await expect(grid.first()).toBeVisible()
  const small = await columnCount(grid)

  await openOptions(page, activate)
  await activate(page.getByRole('menuitem', { name: 'Poster size' }))
  await activate(page.getByRole('menuitem', { name: /^Large/ }))
  await expect(grid.first()).toBeVisible()
  const large = await columnCount(grid)
  expect(large).toBeLessThan(small)

  await openOptions(page, activate)
  await activate(page.getByRole('menuitem', { name: 'View' }))
  await activate(page.getByRole('menuitem', { name: /^Table view/ }))
  const table = libraryRegion(page, 'Movies').getByRole('table')
  await expect(table).toBeVisible()
  await expect(table.getByRole('columnheader', { name: 'Title' })).toBeVisible()
  await expect(table.getByRole('columnheader', { name: 'Year' })).toBeVisible()
  await expect(table.getByRole('columnheader', { name: 'Quality Profile' })).toBeVisible()
  await expect(table.getByRole('columnheader', { name: 'Path' })).toHaveCount(0)
})

test('chip filters narrow the library and restore it', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(cells(page, 'Movies').first()).toBeVisible()

  if (kind === 'phone') {
    const row = page.getByRole('group', { name: 'Filter movies' })
    const scrollable = await row.evaluate((el) => el.scrollWidth > el.clientWidth + 1)
    expect(scrollable).toBe(true)
    const scrolled = await row.evaluate((el) => {
      el.scrollLeft = 120
      return el.scrollLeft
    })
    expect(scrolled).toBeGreaterThan(0)
  }

  // Missing is the ticket's filter, and the set it selects is the set of movies
  // the API reports without files — the developer library's own count, whatever
  // the mock download client has imported by now.
  await activate(chip(page, 'Missing'))
  await expect.poll(async () => missingMatchesApi(page)).toBe(true)
  // The restored set is read through a title that keeps its files.
  await expect(cell(page, 'Movies', 'The Matrix')).toHaveCount(0)
  await activate(chip(page, 'Missing'))
  await expect(cell(page, 'Movies', 'The Matrix').first()).toBeVisible()

  // Unreleased titles never change status, so they give the narrowing a stable
  // pair to check against.
  const movies = await apiGet<LibraryMovie[]>(page, '/movies')
  const unreleased = movies.find((movie) => movie.status === 'unreleased')
  const released = movies.find((movie) => movie.status !== 'unreleased')
  expect(unreleased).toBeDefined()
  expect(released).toBeDefined()
  const unreleasedTitle = unreleased?.title ?? ''
  const releasedTitle = released?.title ?? ''

  await activate(chip(page, 'Unreleased'))
  await expect(cell(page, 'Movies', releasedTitle)).toHaveCount(0)
  await expect(cell(page, 'Movies', unreleasedTitle).first()).toBeVisible()

  await activate(chip(page, 'Unreleased'))
  await expect(cell(page, 'Movies', releasedTitle).first()).toBeVisible()
})

// Other specs add and remove titles in parallel, so the visible set is checked
// against the missing set the API reports rather than against its size.
async function missingMatchesApi(page: Page): Promise<boolean> {
  const visible = await cells(page, 'Movies').allTextContents()
  const movies = await apiGet<LibraryMovie[]>(page, '/movies')
  const missing = movies.filter((movie) => movie.status === 'missing').map((movie) => movie.title)
  return visible.every((text) => missing.some((title) => text.startsWith(title)))
}

test('a poster cell opens its detail', async ({ page, activate }) => {
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await activate(cell(page, 'Movies', 'The Matrix').first())
  await expect(page).toHaveURL(/\/movies\/\d+$/)
  await expect(page.getByRole('button', { name: 'Library' })).toBeVisible()
})

test('search renders library results as poster cells', async ({ page }) => {
  await page.goto('/search?q=matrix')
  const result = page.getByRole('link', { name: /The Matrix/ }).first()
  await expect(result).toBeVisible()
  await expect(result).toHaveAttribute('href', /\/movies\/\d+$/)
  await expect(result.getByRole('img', { name: STATUS_LABEL })).toBeVisible()
})

test('cells scale on pointer down', async ({ page }) => {
  await page.goto('/movies')
  const target = cells(page, 'Movies').first()
  await expect(target).toBeVisible()
  const box = await target.boundingBox()
  expect(box).not.toBeNull()
  if (box === null) {
    return
  }

  // Releasing the button follows the link, so the press is measured while it is
  // held and the release is left to open the detail.
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await expect.poll(async () => scaleOf(target)).toBeLessThan(1)
  await page.mouse.up()
  await expect(page).toHaveURL(/\/movies\/\d+$/)
})

test('rows tint on press and hover only with a fine pointer', async ({ page }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await page.goto('/more')
  const row = page.getByRole('region', { name: 'More' }).getByRole('link', { name: 'Calendar' })
  await expect(row).toBeVisible()
  const idle = await background(row)

  await row.hover()
  const hovered = await background(row)
  if (kind === 'wide') {
    expect(hovered).not.toBe(idle)
  } else {
    expect(hovered).toBe(idle)
  }

  const box = await row.boundingBox()
  expect(box).not.toBeNull()
  if (box === null) {
    return
  }
  await page.mouse.move(box.x + 8, box.y + box.height / 2)
  await page.mouse.down()
  await expect.poll(async () => background(row)).not.toBe(idle)
  expect(await scaleOf(row)).toBe(1)
  await page.mouse.up()
  await expect(page.getByRole('heading', { name: /Calendar/ })).toBeVisible()
})

async function scaleOf(locator: Locator): Promise<number> {
  return locator.evaluate((element) => {
    const transform = globalThis.getComputedStyle(element).transform
    if (transform === 'none') {
      return 1
    }
    return new DOMMatrixReadOnly(transform).a
  })
}

async function background(locator: Locator): Promise<string> {
  return locator.evaluate((element) => globalThis.getComputedStyle(element).backgroundColor)
}
