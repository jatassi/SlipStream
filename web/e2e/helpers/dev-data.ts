import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'

import { apiBase } from './paths'

type HealthItem = {
  name: string
  status: string
  message?: string
}

type HealthResponse = {
  downloadClients: HealthItem[]
  indexers: HealthItem[]
  prowlarr: HealthItem[]
  rootFolders: HealthItem[]
  metadata: HealthItem[]
  storage: HealthItem[]
  import: HealthItem[]
}

type QueueItem = {
  title: string
  releaseName: string
  status: string
  mediaType: string
  movieId?: number
  seriesId?: number
}

function isQueuedOrDownloading(item: QueueItem): boolean {
  return item.status === 'downloading' || item.status === 'queued'
}

type HistoryItem = {
  mediaTitle?: string
  eventType: string
}

type Movie = {
  id: number
  title: string
  status: string
}


async function bearerToken(page: Page): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('slipstream-portal-auth')
    if (!raw) {
      throw new Error('missing portal auth')
    }
    const parsed = JSON.parse(raw) as { state?: { token?: string } }
    const token = parsed.state?.token
    if (!token) {
      throw new Error('missing auth token')
    }
    return token
  })
}

async function apiJson<T>(page: Page, path: string, init?: { method?: string; data?: unknown }): Promise<T> {
  const token = await bearerToken(page)
  const response = await page.request.fetch(`${apiBase}${path}`, {
    method: init?.method ?? 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    data: init?.data,
  })
  if (!response.ok()) {
    throw new Error(`${path} failed: ${response.status()}`)
  }
  return (await response.json()) as T
}

function flattenIssues(health: HealthResponse): HealthItem[] {
  return [
    ...health.downloadClients,
    ...health.indexers,
    ...health.prowlarr,
    ...health.rootFolders,
    ...health.metadata,
    ...health.storage,
    ...health.import,
  ].filter((item) => item.status !== 'ok')
}

export async function apiGet<T>(page: Page, path: string): Promise<T> {
  return apiJson<T>(page, path)
}

export async function listHealthIssues(page: Page): Promise<HealthItem[]> {
  const health = await apiJson<HealthResponse>(page, '/system/health')
  return flattenIssues(health)
}

export async function ensureHealthIssue(page: Page): Promise<HealthItem> {
  const existing = await listHealthIssues(page)
  if (existing[0]) {
    return existing[0]
  }
  await apiJson(page, '/system/health/rootFolders/test', { method: 'POST' })
  const after = await listHealthIssues(page)
  if (!after[0]) {
    throw new Error('root folder health test produced no issues')
  }
  return after[0]
}

export async function listDownloading(page: Page): Promise<QueueItem[]> {
  const data = await apiJson<{ items: QueueItem[] }>(page, '/queue')
  return data.items.filter((item) => item.status === 'downloading')
}

async function listQueuedOrDownloading(page: Page): Promise<QueueItem[]> {
  const data = await apiJson<{ items: QueueItem[] }>(page, '/queue')
  return data.items.filter((item) => isQueuedOrDownloading(item))
}

export type DownloadRef = {
  rowTitle: string
  releaseName: string
  detailTitle: string
}

async function waitForDownloading(page: Page): Promise<QueueItem> {
  await expect
    .poll(
      async () => {
        const active = await listDownloading(page)
        return active[0]?.title ?? ''
      },
      { timeout: 20_000 },
    )
    .not.toEqual('')
  const active = await listDownloading(page)
  if (active.length === 0) {
    throw new Error('queue never entered downloading')
  }
  return active[0]
}

async function detailTitleFor(page: Page, item: QueueItem): Promise<string> {
  if (!item.movieId) {
    return item.title
  }
  const movie = await apiJson<Movie>(page, `/movies/${item.movieId}`)
  return movie.title
}

async function listHistory(page: Page): Promise<HistoryItem[]> {
  const data = await apiJson<{ items: HistoryItem[] }>(page, '/history?pageSize=5')
  return data.items
}

type AutoSearchResult = {
  downloaded?: boolean
  error?: string
}

function pickMovieToSearch(movies: Movie[]): Movie | undefined {
  const rank = ['missing', 'upgradable', 'failed', 'downloading', 'available'] as const
  for (const status of rank) {
    const movie = movies.find((item) => item.status === status)
    if (movie) {
      return movie
    }
  }
}

async function autosearchMissingMovie(page: Page): Promise<void> {
  const movies = await apiJson<Movie[]>(page, '/movies')
  const movie = pickMovieToSearch(movies)
  if (!movie) {
    throw new Error('no searchable movie in the library')
  }
  const token = await bearerToken(page)
  const response = await page.request.fetch(`${apiBase}/autosearch/movie/${movie.id}`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (response.status() === 409) {
    return
  }
  if (!response.ok()) {
    throw new Error(`/autosearch/movie/${movie.id} failed: ${response.status()} ${await response.text()}`)
  }
  const result = (await response.json()) as AutoSearchResult
  if (!result.downloaded) {
    throw new Error(`autosearch did not grab ${movie.title}: ${result.error ?? 'unknown'}`)
  }
}

async function downloadRef(page: Page, item: QueueItem): Promise<DownloadRef> {
  return {
    rowTitle: item.title,
    releaseName: item.releaseName,
    detailTitle: await detailTitleFor(page, item),
  }
}

export async function ensureDownloading(page: Page): Promise<DownloadRef> {
  const existing = await listDownloading(page)
  if (existing[0]) {
    return downloadRef(page, existing[0])
  }
  const pending = await listQueuedOrDownloading(page)
  if (!pending[0]) {
    await autosearchMissingMovie(page)
  }
  return downloadRef(page, await waitForDownloading(page))
}

type Episode = {
  id: number
  status: string
  monitored: boolean
}

// Mock downloads sit in `queued` for two seconds before they start, and a
// queued row carries no pause control, so wait for the running state.
async function grabbed(page: Page, match: (item: QueueItem) => boolean): Promise<QueueItem> {
  const running = (item: QueueItem) => match(item) && item.status === 'downloading'
  await expect
    .poll(
      async () => {
        const data = await apiJson<{ items: QueueItem[] }>(page, '/queue')
        return data.items.filter((item) => running(item)).length
      },
      { timeout: 20_000 },
    )
    .toBeGreaterThan(0)
  const data = await apiJson<{ items: QueueItem[] }>(page, '/queue')
  const item = data.items.find((entry) => running(entry))
  if (!item) {
    throw new Error('grabbed item vanished from the queue')
  }
  return item
}

async function postAutosearch(page: Page, path: string): Promise<void> {
  const token = await bearerToken(page)
  const response = await page.request.fetch(`${apiBase}${path}`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!response.ok() && response.status() !== 409) {
    throw new Error(`${path} failed: ${response.status()} ${await response.text()}`)
  }
}

export async function ensureSeriesDownloading(page: Page, title: string): Promise<DownloadRef> {
  const library = await apiJson<{ id: number; title: string }[]>(page, '/series')
  const series = library.find((entry) => entry.title === title)
  if (!series) {
    throw new Error(`no series titled ${title} in the developer library`)
  }
  const matches = (item: QueueItem) => item.seriesId === series.id
  const queue = await apiJson<{ items: QueueItem[] }>(page, '/queue')
  if (!queue.items.some((item) => matches(item))) {
    const episodes = await apiJson<Episode[]>(page, `/series/${series.id}/episodes`)
    const episode = episodes.find((entry) => entry.monitored && entry.status === 'missing')
    if (!episode) {
      throw new Error(`no missing episode of ${title} to grab`)
    }
    await postAutosearch(page, `/autosearch/episode/${episode.id}`)
  }
  return downloadRef(page, await grabbed(page, matches))
}

export async function addOfflineDownloadClient(page: Page, name: string): Promise<number> {
  const created = await apiJson<{ id: number }>(page, '/downloadclients', {
    method: 'POST',
    data: {
      name,
      type: 'qbittorrent',
      host: '127.0.0.1',
      port: 1,
      enabled: true,
      priority: 99,
      cleanupMode: 'leave',
      importDelaySeconds: 0,
    },
  })
  return created.id
}

export async function removeDownloadClient(page: Page, id: number): Promise<void> {
  const token = await bearerToken(page)
  const response = await page.request.fetch(`${apiBase}/downloadclients/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!response.ok()) {
    throw new Error(`delete download client ${id} failed: ${response.status()}`)
  }
}

// Episode grabs are recorded without a media title, so return the newest entry
// the dashboard's Recent group can actually name.
export async function ensureRecentHistory(page: Page): Promise<HistoryItem> {
  await ensureDownloading(page)
  await expect.poll(async () => {
    const items = await listHistory(page)
    return items.filter((item) => (item.mediaTitle ?? '') !== '').length
  }).toBeGreaterThan(0)
  const history = await listHistory(page)
  const titled = history.find((item) => (item.mediaTitle ?? '') !== '')
  if (!titled) {
    throw new Error('history holds no titled entry')
  }
  return titled
}
