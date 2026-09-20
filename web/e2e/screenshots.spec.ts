import type { Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { findMovie } from './helpers/dev-data'
import { installQueueStub, type StubQueueItem } from './helpers/queue-stub'
import { THEMES, useTheme } from './helpers/theme'

// Baselines are opt-in and exist to catch layout regressions, not to pin pixels;
// the live developer-mode data behind these screens keeps moving, so the queue is
// served from the stub and the tolerance stays generous.
const screenshotOptions = {
  animations: 'disabled' as const,
  maxDiffPixelRatio: 0.08,
}

const MOVIE_TITLE = 'The Matrix'

const STUB_QUEUE: StubQueueItem[] = [
  {
    id: 'shot-downloading',
    title: 'Stub Downloading',
    releaseName: 'Stub.Downloading.2019.1080p.BluRay.x264-STUB',
    mediaType: 'movie',
    status: 'downloading',
    progress: 42,
  },
  {
    id: 'shot-queued',
    title: 'Stub Queued',
    releaseName: 'Stub.Queued.2021.1080p.WEB-DL-STUB',
    mediaType: 'movie',
    status: 'queued',
    progress: 0,
  },
]

type Screen = {
  name: string
  open: (page: Page) => Promise<void>
}

const SCREENS: Screen[] = [
  {
    name: 'dashboard',
    open: async (page) => {
      await installQueueStub(page, STUB_QUEUE)
      await page.goto('/')
      await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
      await expect(page.getByRole('heading', { name: 'Recent' })).toBeVisible()
    },
  },
  {
    name: 'library',
    open: async (page) => {
      await page.goto('/movies')
      await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
      await expect(page.getByRole('link', { name: new RegExp(MOVIE_TITLE) })).toBeVisible()
    },
  },
  {
    name: 'detail',
    open: async (page) => {
      await page.goto('/movies')
      const movie = await findMovie(page, MOVIE_TITLE)
      await page.goto(`/movies/${movie.id}`)
      await expect(page.getByRole('heading', { name: movie.title, level: 1 })).toBeVisible()
    },
  },
  {
    name: 'activity',
    open: async (page) => {
      await installQueueStub(page, STUB_QUEUE)
      await page.goto('/downloads')
      await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
      await expect(page.getByText('Stub Downloading')).toBeVisible()
    },
  },
  {
    name: 'settings',
    open: async (page) => {
      await page.goto('/settings/media/quality-profiles')
      await expect(page.getByRole('heading', { name: 'Quality Profiles' })).toBeVisible()
    },
  },
]

for (const screen of SCREENS) {
  for (const theme of THEMES) {
    test(`${screen.name} screenshot comparison (${theme})`, async ({ page }) => {
      test.skip(process.env.PLAYWRIGHT_SCREENSHOTS !== '1', 'screenshot comparison is opt-in')
      await useTheme(page, theme)
      await screen.open(page)
      await expect(page).toHaveScreenshot(`${screen.name}-${theme}.png`, screenshotOptions)
    })
  }
}
