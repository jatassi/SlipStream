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
  status: string
  movieId?: number
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

export async function ensureDownloading(page: Page): Promise<DownloadRef> {
  const existing = await listDownloading(page)
  if (existing[0]) {
    return { rowTitle: existing[0].title, detailTitle: await detailTitleFor(page, existing[0]) }
  }
  const pending = await listQueuedOrDownloading(page)
  if (!pending[0]) {
    await autosearchMissingMovie(page)
  }
  const item = await waitForDownloading(page)
  return { rowTitle: item.title, detailTitle: await detailTitleFor(page, item) }
}

export async function ensureRecentHistory(page: Page): Promise<HistoryItem> {
  await ensureDownloading(page)
  await expect.poll(async () => {
    const items = await listHistory(page)
    return items.length
  }).toBeGreaterThan(0)
  const history = await listHistory(page)
  if (history.length === 0) {
    throw new Error('history stayed empty after autosearch')
  }
  return history[0]
}
