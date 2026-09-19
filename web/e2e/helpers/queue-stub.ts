import type { Page, WebSocketRoute } from '@playwright/test'

// The developer-mode download client only hands out the states it reaches on its
// own clock: an item is queued for two seconds and completes after five minutes.
// Two checks need a state held still — queued and importing, and two progress
// values pushed a moment apart — so those serve the queue from here instead.
// The real socket is intercepted as well: `queue:state` frames write straight
// into the query cache and would otherwise overwrite the served items.

export type StubQueueItem = {
  id: string
  title: string
  releaseName: string
  mediaType: 'movie' | 'series'
  status: 'queued' | 'downloading' | 'paused' | 'completed'
  progress: number
  season?: number
  episode?: number
}

export type QueueStub = {
  push: (items: StubQueueItem[]) => void
}

const CLIENT_ID = 99
const SIZE = 12 * 1024 * 1024 * 1024
const SPEED = 5 * 1024 * 1024

function toQueueItem(item: StubQueueItem) {
  const active = item.status === 'downloading'
  return {
    id: item.id,
    clientId: CLIENT_ID,
    clientName: 'Stub Client',
    clientType: 'qbittorrent',
    title: item.title,
    releaseName: item.releaseName,
    mediaType: item.mediaType,
    status: item.status,
    progress: item.progress,
    size: SIZE,
    downloadedSize: Math.round((SIZE * item.progress) / 100),
    downloadSpeed: active ? SPEED : 0,
    eta: active ? 320 : 0,
    attributes: [],
    season: item.season ?? 0,
    episode: item.episode ?? 0,
    downloadPath: '/mock/downloads',
  }
}

function queuePayload(items: StubQueueItem[]): string {
  return JSON.stringify({ items: items.map((item) => toQueueItem(item)) })
}

export async function installQueueStub(page: Page, initial: StubQueueItem[]): Promise<QueueStub> {
  let items = initial
  const socket: { current: WebSocketRoute | null } = { current: null }

  await page.routeWebSocket(/\/ws(\?.*)?$/, (ws) => {
    socket.current = ws
  })

  await page.route('**/api/v1/queue', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: queuePayload(items),
    })
  })

  return {
    push: (next: StubQueueItem[]) => {
      items = next
      socket.current?.send(
        JSON.stringify({
          type: 'queue:state',
          payload: JSON.parse(queuePayload(items)) as unknown,
          timestamp: new Date().toISOString(),
        }),
      )
    },
  }
}
