import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { type ShellKind, shellKind } from './helpers/activate'
import {
  apiGet,
  createScratchMovie,
  deleteMovieIfPresent,
  findMovie,
  findSeries,
  type MovieDetail,
} from './helpers/dev-data'
import { installQueueStub } from './helpers/queue-stub'

const MOVIE_TITLE = 'The Matrix'
const SERIES_TITLE: Record<ShellKind, string> = {
  phone: 'Breaking Bad',
  wide: 'Game of Thrones',
}
const SCRATCH_TITLE: Record<ShellKind, string> = {
  phone: 'Scratch Movie Phone',
  wide: 'Scratch Movie Wide',
}

const STATUS_LABEL: Record<string, string> = {
  available: 'Available',
  missing: 'Missing',
  downloading: 'Downloading',
  upgradable: 'Upgradable',
  unreleased: 'Unreleased',
  failed: 'Failed',
}

// The presenter renders its actions as sheet buttons on phones and as menu
// items in a dropdown on wide screens.
function action(page: Page, kind: ShellKind, name: string) {
  return kind === 'wide'
    ? page.getByRole('menuitem', { name, exact: true })
    : page.getByRole('button', { name, exact: true })
}

async function openMovie(page: Page, title = MOVIE_TITLE): Promise<MovieDetail> {
  await page.goto('/movies')
  const movie = await findMovie(page, title)
  await page.goto(`/movies/${movie.id}`)
  await expect(page.getByRole('heading', { name: movie.title, level: 1 })).toBeVisible()
  return movie
}

test('movie detail shows artwork, facts and a status pill', async ({ page }) => {
  const movie = await openMovie(page)
  const region = page.getByRole('region', { name: movie.title })
  await expect(region.getByRole('img', { name: movie.title }).first()).toBeVisible()
  await expect(page.getByText('Movie', { exact: true })).toBeVisible()
  if (movie.year !== undefined) {
    await expect(page.getByText(new RegExp(`${movie.year}`)).first()).toBeVisible()
  }
  await expect(page.getByText(STATUS_LABEL[movie.status], { exact: true }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Search', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Auto Search' })).toBeVisible()
})

test('series detail shows the season count in place of a runtime', async ({ page }, testInfo) => {
  await page.goto('/series')
  const series = await findSeries(page, SERIES_TITLE[shellKind(testInfo.project.name)])
  await page.goto(`/series/${series.id}`)
  await expect(page.getByRole('heading', { name: series.title, level: 1 })).toBeVisible()
  const seasons = series.seasons.filter((season) => season.seasonNumber > 0).length
  const label = seasons === 1 ? '1 Season' : `${seasons} Seasons`
  await expect(page.getByText(new RegExp(label))).toBeVisible()
})

// Each project toggles a movie of its own: the two run against one backend.
test('monitored pill toggles and the state survives a reload', async ({ page, activate }, testInfo) => {
  const movie = await openMovie(page, await monitorTitle(page, shellKind(testInfo.project.name)))
  const first = movie.monitored ? 'Monitored' : 'Unmonitored'
  const second = movie.monitored ? 'Unmonitored' : 'Monitored'

  await activate(page.getByRole('button', { name: first }))
  await expect(page.getByRole('button', { name: second })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: second })).toBeVisible()

  await activate(page.getByRole('button', { name: second }))
  await expect(page.getByRole('button', { name: first })).toBeVisible()
})

async function monitorTitle(page: Page, kind: ShellKind): Promise<string> {
  await page.goto('/movies')
  const movies = await apiGet<{ title: string }[]>(page, '/movies')
  const titles = movies
    .map((movie) => movie.title)
    .filter((title) => title !== MOVIE_TITLE && !title.startsWith('Scratch Movie'))
    .toSorted()
  const title: string | undefined = titles.at(kind === 'phone' ? 0 : 1)
  if (title === undefined) {
    throw new Error('the developer library has too few movies for the monitor check')
  }
  return title
}

test('auto search shows the searching state and an error that can be dismissed', async ({
  page,
  activate,
}) => {
  const movie = await openMovie(page)
  // Held open so the searching state is observable: developer mode's own delay
  // only applies once the dev-mode flag has loaded.
  await page.route(`**/api/v1/autosearch/movie/${movie.id}`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 1500))
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ found: false, downloaded: false, upgraded: false }),
    })
  })

  await activate(page.getByRole('button', { name: 'Auto Search' }))
  await expect(page.getByRole('button', { name: 'Searching...' })).toBeVisible()
  const dismiss = page.getByRole('button', { name: /Not Found — dismiss/ })
  await expect(dismiss).toBeVisible({ timeout: 20_000 })
  await activate(dismiss)
  await expect(page.getByRole('button', { name: 'Auto Search' })).toBeVisible()
})

test('a finished download flashes the completed state in the pill row', async ({ page, activate }) => {
  await page.goto('/movies')
  const movie = await findMovie(page, MOVIE_TITLE)
  const stub = await installQueueStub(page, [
    {
      id: 'stub-detail',
      title: movie.title,
      releaseName: 'Stub.Detail.1999.1080p.BluRay-STUB',
      mediaType: 'movie',
      status: 'downloading',
      progress: 64,
      movieId: movie.id,
    },
  ])
  await page.goto(`/movies/${movie.id}`)
  await expect(page.getByText(/Downloading/).first()).toBeVisible()

  stub.push([])
  const downloaded = page.getByRole('button', { name: 'Downloaded — dismiss' })
  await expect(downloaded).toBeVisible()
  await activate(downloaded)
  await expect(page.getByRole('button', { name: 'Search', exact: true })).toBeVisible()
})

test('a long overview clamps to three lines and expands on tap', async ({ page, activate }) => {
  await openMovie(page)
  const overview = page.getByRole('region', { name: 'Overview' })
  const expand = page.getByRole('button', { name: 'Expand overview' })
  await expect(expand).toBeVisible()
  await expect.poll(async () => lineClamp(overview)).toBe('3')

  await activate(expand)
  const collapse = page.getByRole('button', { name: 'Collapse overview' })
  await expect(collapse).toBeVisible()
  await expect.poll(async () => lineClamp(overview)).toBe('none')

  await activate(collapse)
  await expect.poll(async () => lineClamp(overview)).toBe('3')
})

async function lineClamp(overview: Locator): Promise<string> {
  return overview.locator('span').first().evaluate((el) => globalThis.getComputedStyle(el).webkitLineClamp)
}

test('File shows version slots and the quality profile, Details shows metadata', async ({ page }) => {
  await page.goto('/movies')
  const movie = await findMovie(page, MOVIE_TITLE)
  await stubVersionSlots(page, movie.id)
  await page.goto(`/movies/${movie.id}`)
  await expect(page.getByRole('heading', { name: movie.title, level: 1 })).toBeVisible()

  const file = page.getByRole('region', { name: 'File' })
  await expect(file.getByText('Quality Profile')).toBeVisible()
  await expect(file.getByText('Standard Slot')).toBeVisible()
  await expect(file.getByText('4K Slot')).toBeVisible()

  const details = page.getByRole('region', { name: 'Details' })
  await expect(details.getByText('Year')).toBeVisible()
  if (movie.year !== undefined) {
    await expect(details.getByText(String(movie.year), { exact: true })).toBeVisible()
  }
})

test('a season group collapses and expands and its episodes offer a search action', async ({
  page,
  activate,
}, testInfo) => {
  await page.goto('/series')
  const series = await findSeries(page, SERIES_TITLE[shellKind(testInfo.project.name)])
  await page.goto(`/series/${series.id}`)
  await expect(page.getByRole('heading', { name: series.title, level: 1 })).toBeVisible()

  const seasons = page.getByRole('region', { name: 'Seasons' })
  const trigger = seasons.getByRole('button', { name: /Season 1/ })
  await expect(trigger).toBeVisible()
  const episode = seasons.getByText(/^1\. /).first()

  await activate(trigger)
  await expect(episode).toBeVisible()
  await expect(seasons.getByRole('img').first()).toBeVisible()
  await expect(seasons.getByRole('button', { name: 'Manual Search' }).first()).toBeVisible()

  await activate(trigger)
  await expect(episode).toBeHidden()
})

test('the trailing menu opens Edit as a sheet that can be dragged away', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await openMovie(page)
  await activate(page.getByRole('button', { name: 'Movie actions' }))
  await activate(action(page, 'phone', 'Edit'))

  const sheet = page.getByRole('dialog').filter({ hasText: 'Edit Movie' })
  await expect(sheet).toBeVisible()
  await expect(sheet.getByText('Quality Profile')).toBeVisible()

  const box = await sheet.boundingBox()
  if (!box) {
    throw new Error('the edit sheet has no box to drag')
  }
  await dragDown(page, { x: box.x + box.width / 2, y: box.y + 14 }, 420)

  await expect(sheet).toBeHidden()
})

test('Edit opens a dialog with the same form on wide screens', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await openMovie(page)
  await activate(page.getByRole('button', { name: 'Movie actions' }))
  await activate(action(page, 'wide', 'Edit'))

  const dialog = page.getByRole('dialog').filter({ hasText: 'Edit Movie' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('Quality Profile')).toBeVisible()
  await activate(dialog.getByRole('button', { name: 'Cancel' }))
  await expect(dialog).toBeHidden()
})

test('Delete confirms and returns to the library without the item', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const title = SCRATCH_TITLE[kind]
  await page.goto('/movies')
  const scratch = await createScratchMovie(page, title)

  try {
    await page.goto(`/movies/${scratch.id}`)
    await expect(page.getByRole('heading', { name: title, level: 1 })).toBeVisible()
    await activate(page.getByRole('button', { name: 'Movie actions' }))
    await activate(action(page, kind, 'Delete'))
    await expect(page.getByText('Delete movie?')).toBeVisible()
    await activate(page.getByRole('button', { name: 'Delete', exact: true }).last())

    await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
    await expect(page.getByRole('link', { name: new RegExp(title) })).toHaveCount(0)
  } finally {
    await deleteMovieIfPresent(page, scratch.id)
  }
})

async function stubVersionSlots(page: Page, movieId: number): Promise<void> {
  await page.route('**/api/v1/slots/settings', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        enabled: true,
        dryRunCompleted: true,
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      }),
    })
  })
  await page.route('**/api/v1/slots', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        slot(1, 'Standard Slot'),
        slot(2, '4K Slot'),
      ]),
    })
  })
  await page.route(`**/api/v1/slots/movies/${movieId}/status`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        mediaType: 'movie',
        mediaId: movieId,
        status: 'missing',
        filledSlots: 0,
        emptySlots: 2,
        monitoredSlots: 2,
        slotStatuses: [
          slotStatus(1, 'Standard Slot'),
          slotStatus(2, '4K Slot'),
        ],
      }),
    })
  })
}

function slot(id: number, name: string) {
  return {
    id,
    slotNumber: id,
    name,
    enabled: true,
    qualityProfileId: null,
    displayOrder: id,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    rootFolders: {},
  }
}

function slotStatus(slotId: number, slotName: string) {
  return {
    slotId,
    slotNumber: slotId,
    slotName,
    monitored: true,
    status: 'missing',
    profileCutoff: 1,
  }
}

// vaul ignores a drag that starts within 500ms of the sheet opening, so the
// gesture waits for the entrance to settle before pulling the sheet down.
async function dragDown(page: Page, from: { x: number; y: number }, distance: number): Promise<void> {
  await page.waitForTimeout(700)
  await page.mouse.move(from.x, from.y)
  await page.mouse.down()
  for (let step = 1; step <= 8; step += 1) {
    await page.mouse.move(from.x, from.y + (distance / 8) * step)
  }
  await page.mouse.up()
}
