import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import type { ShellKind } from './helpers/activate'
import { shellKind } from './helpers/activate'
import { apiGet, deleteMovieIfPresent, setMoviesMonitored } from './helpers/dev-data'
import { applySafeArea, primaryNav } from './helpers/shell'

type LibraryMovie = { id: number; title: string; monitored: boolean }
type Named = { name: string }

// A title the developer metadata provider knows and the developer library does
// not hold. Each project adds its own so the two runs never collide, and the
// added movie is left in the library afterwards.
const ADD_TITLES: Record<ShellKind, string> = {
  phone: 'Ford v Ferrari',
  wide: 'Everything Everywhere All at Once',
}

// Titles whose monitored state no other spec reads; detail.spec takes the first
// two of the library's sorted titles, which these are not.
const BULK_TITLES: Record<ShellKind, string[]> = {
  phone: ['Fight Club', 'Pulp Fiction'],
  wide: ['Inception', 'Dune: Part Two'],
}

function cells(page: Page): Locator {
  return page.getByRole('list', { name: 'Movies', exact: true }).getByRole('link')
}

function optionsButton(page: Page): Locator {
  return page.getByRole('button', { name: 'Options', exact: true })
}

function toolbar(page: Page): Locator {
  return page.getByRole('toolbar', { name: 'Selection actions' })
}

async function openOptions(page: Page, activate: (locator: Locator) => Promise<void>): Promise<void> {
  await activate(optionsButton(page))
}

type Shell = {
  page: Page
  activate: (locator: Locator) => Promise<void>
  kind: ShellKind
}

async function chooseOption(shell: Shell, name: string | RegExp): Promise<void> {
  const role = shell.kind === 'phone' ? 'button' : 'menuitem'
  await shell.activate(shell.page.getByRole(role, { name }))
}

// The developer library's movies carry no release date, no recorded size and one
// shared date added, so the title order is the only one its data can reorder.
// Two titles are compared rather than the first cell, since other specs add and
// remove titles around this one.
async function precedes(page: Page, first: string, second: string): Promise<boolean> {
  await expect(cells(page).first()).toBeVisible()
  const titles = await cells(page).allTextContents()
  const a = titles.findIndex((text) => text.startsWith(first))
  const b = titles.findIndex((text) => text.startsWith(second))
  return a !== -1 && b !== -1 && a < b
}

async function enterEditMode(shell: Shell): Promise<void> {
  await openOptions(shell.page, shell.activate)
  await chooseOption(shell, 'Select movies')
  await expect(toolbar(shell.page)).toBeVisible()
}

test('the top bar carries one options action that sorts the grid', async ({
  page,
  activate,
}, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const shell: Shell = { page, activate, kind }
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(optionsButton(page)).toHaveCount(1)
  // The phone sheet hides the page behind it, so the grid is read first.
  expect(await precedes(page, 'Barbie', 'The Matrix')).toBe(true)

  await openOptions(page, activate)
  if (kind === 'phone') {
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    await expect(page.getByRole('menu')).toHaveCount(0)
  } else {
    await expect(page.getByRole('menu')).toBeVisible()
    await expect(page.getByRole('dialog')).toHaveCount(0)
  }

  await chooseOption(shell, /^Reverse order/)
  await expect.poll(async () => precedes(page, 'The Matrix', 'Barbie')).toBe(true)

  await page.reload()
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  expect(await precedes(page, 'The Matrix', 'Barbie')).toBe(true)

  await openOptions(page, activate)
  await chooseOption(shell, 'Sort by Title')
  await chooseOption(shell, 'Date Added')
  await expect(page.getByText('Today', { exact: true })).toBeVisible()

  await page.reload()
  await expect(page.getByText('Today', { exact: true })).toBeVisible()
  await openOptions(page, activate)
  await expect(page.getByText('Sort by Date Added')).toBeVisible()
})

test('view and column options belong to the wide menu only', async ({
  page,
  activate,
}, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const shell: Shell = { page, activate, kind }
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await openOptions(page, activate)

  if (kind === 'wide') {
    await expect(page.getByRole('menuitem', { name: 'View' })).toBeVisible()
    await expect(page.getByRole('menuitem', { name: 'Poster size' })).toBeVisible()
    await chooseOption(shell, 'View')
    await chooseOption(shell, /^Table view/)
    await openOptions(page, activate)
    await expect(page.getByRole('menuitem', { name: 'Columns' })).toBeVisible()
    await chooseOption(shell, 'Columns')
    await expect(page.getByRole('menuitem', { name: /^Year/ })).toBeVisible()
    await page.keyboard.press('Escape')
    await openOptions(page, activate)
    await chooseOption(shell, 'View')
    await chooseOption(shell, /^Grid view/)
    return
  }

  await expect(page.getByRole('button', { name: /^Sort by/ })).toBeVisible()
  await expect(page.getByRole('button', { name: 'View' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Columns' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: /view$/ })).toHaveCount(0)
})

test('edit mode selects cells and bulk unmonitors them', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const shell: Shell = { page, activate, kind }
  // Two library titles no other spec changes the monitored state of, one pair
  // per project so the two runs never fight over the same movies.
  const names = BULK_TITLES[kind]
  await page.goto('/movies')
  const library = await apiGet<LibraryMovie[]>(page, '/movies')
  const picked = names.map((name) => {
    const movie = library.find((item) => item.title === name)
    if (movie === undefined) {
      throw new Error(`no movie titled ${name} in the developer library`)
    }
    return movie
  })

  await enterEditMode(shell)
  const unmonitor = toolbar(page).getByRole('button', { name: 'Unmonitor selected movies' })
  await expect(unmonitor).toBeDisabled()

  for (const name of names) {
    const mark = page.getByRole('button', { name: `Select ${name}`, exact: true })
    await expect(mark).toHaveAttribute('aria-pressed', 'false')
    await activate(mark)
    await expect(mark).toHaveAttribute('aria-pressed', 'true')
  }
  await expect(toolbar(page).getByText('2 selected')).toBeVisible()
  await expect(unmonitor).toBeEnabled()

  await activate(unmonitor)
  await expect(toolbar(page)).toHaveCount(0)
  for (const movie of picked) {
    await page.goto(`/movies/${movie.id}`)
    await expect(page.getByRole('button', { name: 'Unmonitored' })).toBeVisible()
  }

  await setMoviesMonitored(
    page,
    picked.map((movie) => movie.id),
    true,
  )
})

test('leaving edit mode hides the selection toolbar', async ({ page, activate }, testInfo) => {
  const shell: Shell = { page, activate, kind: shellKind(testInfo.project.name) }
  await page.goto('/movies')
  await expect(cells(page).first()).toBeVisible()
  await enterEditMode(shell)
  await expect(cells(page)).toHaveCount(0)

  await activate(page.getByRole('button', { name: 'Done', exact: true }))
  await expect(toolbar(page)).toHaveCount(0)
  await expect(cells(page).first()).toBeVisible()
})

test('the phone edit toolbar clears the tab bar and the safe area', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/movies')
  await expect(cells(page).first()).toBeVisible()
  await applySafeArea(page)
  await enterEditMode({ page, activate, kind: 'phone' })

  const bar = await toolbar(page).boundingBox()
  const tabs = await primaryNav(page).boundingBox()
  expect(bar).not.toBeNull()
  expect(tabs).not.toBeNull()
  if (bar === null || tabs === null) {
    return
  }
  expect(bar.y).toBeGreaterThan(0)
  expect(bar.y + bar.height).toBeLessThanOrEqual(tabs.y + 1)
})

test('add searches the provider and puts the title in the grid', async ({
  page,
  activate,
}, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const shell: Shell = { page, activate, kind }
  const title = ADD_TITLES[kind]
  await page.goto('/movies')
  const library = await apiGet<LibraryMovie[]>(page, '/movies')
  for (const movie of library.filter((item) => item.title === title)) {
    await deleteMovieIfPresent(page, movie.id)
  }

  await activate(page.getByRole('link', { name: 'Add Movie' }))
  await expect(page).toHaveURL(/\/search/)

  await page.goto(`/search?q=${encodeURIComponent(title)}`)
  const result = page.getByRole('button', { name: /^Add/ }).first()
  await expect(result).toBeVisible()
  await activate(result)

  if (kind === 'phone') {
    await expect(page.getByRole('heading', { name: 'Add Movie' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Library', exact: true })).toBeVisible()
  } else {
    await expect(page.getByRole('dialog', { name: 'Add Movie' })).toBeVisible()
  }
  await expect(page.getByRole('heading', { name: title })).toBeVisible()

  const folders = await apiGet<Named[]>(page, '/rootfolders?mediaType=movie')
  const profiles = await apiGet<Named[]>(page, '/qualityprofiles?mediaType=movie')
  await pickOption(shell, 'Root Folder', folders[0].name)
  await pickOption(shell, 'Quality Profile', profiles[0].name)

  await activate(page.getByRole('button', { name: 'Add Movie', exact: true }))
  await expect(page).toHaveURL(/\/movies\/\d+$/)

  await page.goto('/movies')
  await expect(page.getByRole('link', { name: new RegExp(title) }).first()).toBeVisible()
})

// A closed Base UI select keeps its options mounted, so the option is taken by
// its own name rather than by position among every mounted option.
async function pickOption(shell: Shell, field: string, option: string): Promise<void> {
  await shell.activate(shell.page.getByRole('combobox', { name: field }))
  await shell.activate(shell.page.getByRole('option', { name: option, exact: true }))
}
