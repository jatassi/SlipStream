import { create } from 'zustand'

import type { PortalDownload, Request } from '@/types/portal'
import type { QueueItem } from '@/types/queue'

type MatchInfo = {
  requestId: number
  requestTitle: string
  requestMediaId?: number
  tmdbId?: number
  tvdbId?: number
}

type PortalDownloadsState = {
  // Raw queue state from WebSocket
  queue: QueueItem[]
  // Cached matches: queueItemId -> match info
  matches: Map<string, MatchInfo>
  // Last known requests for matching
  userRequests: Request[]
  // Timestamp of last WebSocket queue update (for fallback polling)
  lastUpdateTime: number

  // Actions
  setQueue: (queue: QueueItem[]) => void
  setUserRequests: (requests: Request[]) => void
  getMatchedDownloads: () => PortalDownload[]
}

export const usePortalDownloadsStore = create<PortalDownloadsState>((set, get) => ({
  queue: [],
  matches: new Map(),
  userRequests: [],
  lastUpdateTime: 0,

  setQueue: (queue) => {
    const items = queue
    const state = get()
    const newMatches = new Map(state.matches)

    console.log('[PortalDownloads] setQueue called', {
      queueLength: items.length,
      userRequestsLength: state.userRequests.length,
      existingMatchesCount: state.matches.size,
    })

    // Match any new queue items that we haven't seen before
    for (const item of items) {
      if (!newMatches.has(item.id)) {
        const match = findMatchingRequest(item, state.userRequests)
        if (match) {
          console.log('[PortalDownloads] New match found', {
            itemId: item.id,
            itemTitle: item.title,
            requestId: match.requestId,
          })
          newMatches.set(item.id, match)
        }
      }
    }

    // Clean up matches for queue items that no longer exist
    const currentIds = new Set(items.map((item) => item.id))
    for (const id of newMatches.keys()) {
      if (!currentIds.has(id)) {
        newMatches.delete(id)
      }
    }

    console.log('[PortalDownloads] setQueue complete', { newMatchesCount: newMatches.size })
    set({ queue: items, matches: newMatches, lastUpdateTime: Date.now() })
  },

  setUserRequests: (requests) => {
    const state = get()
    const newMatches = new Map<string, MatchInfo>()

    console.log('[PortalDownloads] setUserRequests called', {
      requestsLength: requests.length,
      queueLength: state.queue.length,
    })

    // Re-match all queue items with new requests
    for (const item of state.queue) {
      const match = findMatchingRequest(item, requests)
      if (match) {
        console.log('[PortalDownloads] Match found on request update', {
          itemId: item.id,
          itemTitle: item.title,
          requestId: match.requestId,
        })
        newMatches.set(item.id, match)
      }
    }

    console.log('[PortalDownloads] setUserRequests complete', { newMatchesCount: newMatches.size })
    set({ userRequests: requests, matches: newMatches })
  },

  getMatchedDownloads: () => {
    const state = get()
    const downloads: PortalDownload[] = []

    for (const item of state.queue) {
      const match = state.matches.get(item.id)
      if (match) {
        downloads.push({
          id: item.id,
          clientId: item.clientId,
          clientName: item.clientName,
          title: item.title,
          mediaType: item.mediaType,
          status: item.status,
          progress: item.progress,
          size: item.size,
          downloadedSize: item.downloadedSize,
          downloadSpeed: item.downloadSpeed,
          eta: item.eta,
          season: item.season,
          episode: item.episode,
          movieId: item.movieId,
          seriesId: item.seriesId,
          seasonNumber: item.seasonNumber,
          isSeasonPack: item.isSeasonPack,
          requestId: match.requestId,
          requestTitle: match.requestTitle,
          requestMediaId: match.requestMediaId,
          tmdbId: match.tmdbId,
          tvdbId: match.tvdbId,
        })
      }
    }

    return downloads
  },
}))

// Normalize title for fuzzy matching (lowercase, remove punctuation, collapse spaces)
function normalizeTitle(title: string): string {
  return title
    .toLowerCase()
    .replaceAll(/[._-]/g, ' ')
    .replaceAll(/[^a-z0-9\s]/g, '')
    .replaceAll(/\s+/g, ' ')
    .trim()
}

function findMatchingRequest(item: QueueItem, requests: Request[]): MatchInfo | null {
  // First pass: match by internal media ID (most reliable)
  for (const req of requests) {
    if (req.mediaId !== null) {
      if (req.mediaType === 'movie' && item.movieId === req.mediaId) {
        return {
          requestId: req.id,
          requestTitle: req.title,
          requestMediaId: req.mediaId,
          tmdbId: req.tmdbId ?? undefined,
          tvdbId: req.tvdbId ?? undefined,
        }
      }
      if (
        (req.mediaType === 'series' || req.mediaType === 'season') &&
        item.seriesId === req.mediaId
      ) {
        return {
          requestId: req.id,
          requestTitle: req.title,
          requestMediaId: req.mediaId,
          tmdbId: req.tmdbId ?? undefined,
          tvdbId: req.tvdbId ?? undefined,
        }
      }
    }
  }

  // Second pass: fallback to title matching for requests without mediaId yet
  // This handles the race condition where auto-approve is still processing
  // (request returned to frontend before mediaId is set)
  const itemTitleNorm = normalizeTitle(item.title)
  for (const req of requests) {
    // Match requests that don't have mediaId yet (pending or just approved)
    if (req.mediaId === null) {
      const reqTitleNorm = normalizeTitle(req.title)
      // Check if the queue item title contains the request title
      if (
        reqTitleNorm.length > 0 &&
        (itemTitleNorm.startsWith(reqTitleNorm) || itemTitleNorm.includes(reqTitleNorm))
      ) {
        // Also verify media type matches
        const typeMatches =
          (req.mediaType === 'movie' && item.mediaType === 'movie') ||
          ((req.mediaType === 'series' || req.mediaType === 'season') &&
            item.mediaType === 'series')
        if (typeMatches) {
          return {
            requestId: req.id,
            requestTitle: req.title,
            requestMediaId: undefined,
            tmdbId: req.tmdbId ?? undefined,
            tvdbId: req.tvdbId ?? undefined,
          }
        }
      }
    }
  }

  return null
}
