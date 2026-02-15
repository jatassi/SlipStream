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
  queue: QueueItem[]
  matches: Map<string, MatchInfo>
  userRequests: Request[]
  lastUpdateTime: number

  setQueue: (queue: QueueItem[]) => void
  setUserRequests: (requests: Request[]) => void
  getMatchedDownloads: () => PortalDownload[]
}

function buildMatchInfo(req: Request, mediaId: number | undefined): MatchInfo {
  return {
    requestId: req.id,
    requestTitle: req.title,
    requestMediaId: mediaId,
    tmdbId: req.tmdbId ?? undefined,
    tvdbId: req.tvdbId ?? undefined,
  }
}

function normalizeTitle(title: string): string {
  return title
    .toLowerCase()
    .replaceAll(/[._-]/g, ' ')
    .replaceAll(/[^a-z0-9\s]/g, '')
    .replaceAll(/\s+/g, ' ')
    .trim()
}

function mediaTypesMatch(reqType: string, itemType: string): boolean {
  if (reqType === 'movie') {
    return itemType === 'movie'
  }
  return (reqType === 'series' || reqType === 'season') && itemType === 'series'
}

function matchByMediaId(item: QueueItem, req: Request): MatchInfo | null {
  if (req.mediaId === null) {
    return null
  }
  if (req.mediaType === 'movie' && item.movieId === req.mediaId) {
    return buildMatchInfo(req, req.mediaId)
  }
  if (mediaTypesMatch(req.mediaType, 'series') && item.seriesId === req.mediaId) {
    return buildMatchInfo(req, req.mediaId)
  }
  return null
}

function matchByTitle(item: QueueItem, req: Request, itemTitleNorm: string): MatchInfo | null {
  if (req.mediaId !== null) {
    return null
  }
  if (!mediaTypesMatch(req.mediaType, item.mediaType)) {
    return null
  }

  const reqTitleNorm = normalizeTitle(req.title)
  if (reqTitleNorm.length === 0) {
    return null
  }

  const titleMatches =
    itemTitleNorm.startsWith(reqTitleNorm) || itemTitleNorm.includes(reqTitleNorm)
  if (!titleMatches) {
    return null
  }

  return buildMatchInfo(req, undefined)
}

function findMatchingRequest(item: QueueItem, requests: Request[]): MatchInfo | null {
  for (const req of requests) {
    const idMatch = matchByMediaId(item, req)
    if (idMatch) {
      return idMatch
    }
  }

  const itemTitleNorm = normalizeTitle(item.title)
  for (const req of requests) {
    const titleMatch = matchByTitle(item, req, itemTitleNorm)
    if (titleMatch) {
      return titleMatch
    }
  }

  return null
}

function matchNewQueueItems(
  items: QueueItem[],
  existing: Map<string, MatchInfo>,
  requests: Request[],
): Map<string, MatchInfo> {
  const newMatches = new Map(existing)

  for (const item of items) {
    if (newMatches.has(item.id)) {
      continue
    }
    const match = findMatchingRequest(item, requests)
    if (match) {
      newMatches.set(item.id, match)
    }
  }

  const currentIds = new Set(items.map((item) => item.id))
  for (const id of newMatches.keys()) {
    if (!currentIds.has(id)) {
      newMatches.delete(id)
    }
  }

  return newMatches
}

function matchAllQueueItems(
  items: QueueItem[],
  requests: Request[],
): Map<string, MatchInfo> {
  const newMatches = new Map<string, MatchInfo>()
  for (const item of items) {
    const match = findMatchingRequest(item, requests)
    if (match) {
      newMatches.set(item.id, match)
    }
  }
  return newMatches
}

function toPortalDownload(item: QueueItem, match: MatchInfo): PortalDownload {
  return {
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
  }
}

export const usePortalDownloadsStore = create<PortalDownloadsState>((set, get) => ({
  queue: [],
  matches: new Map(),
  userRequests: [],
  lastUpdateTime: 0,

  setQueue: (queue) => {
    const state = get()
    const newMatches = matchNewQueueItems(queue, state.matches, state.userRequests)
    set({ queue, matches: newMatches, lastUpdateTime: Date.now() })
  },

  setUserRequests: (requests) => {
    const state = get()
    const newMatches = matchAllQueueItems(state.queue, requests)
    set({ userRequests: requests, matches: newMatches })
  },

  getMatchedDownloads: () => {
    const state = get()
    const downloads: PortalDownload[] = []
    for (const item of state.queue) {
      const match = state.matches.get(item.id)
      if (match) {
        downloads.push(toPortalDownload(item, match))
      }
    }
    return downloads
  },
}))
