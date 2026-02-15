import type { GrabRequest, TorrentInfo } from '@/types'

import type { SortColumn } from './search-modal-types'
import { RESOLUTION_ORDER } from './search-modal-types'

type DialogTitleParams = {
  seriesTitle?: string
  season?: number
  episode?: number
  mediaTitle: string
}

export function buildDialogTitle(params: DialogTitleParams): string {
  const { seriesTitle, season, episode, mediaTitle } = params

  if (seriesTitle && season !== undefined && episode !== undefined) {
    return `Search: ${seriesTitle} S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
  }
  if (seriesTitle && season !== undefined) {
    return `Search: ${seriesTitle} Season ${season}`
  }
  if (mediaTitle) {
    return `Search: ${mediaTitle}`
  }
  return 'Search Releases'
}

type CompareGetter = (r: TorrentInfo) => number | string

const COMPARATORS: Record<SortColumn, CompareGetter> = {
  score: (r) => r.score ?? 0,
  title: (r) => r.title,
  quality: (r) => RESOLUTION_ORDER[r.quality ?? ''] ?? -1,
  slot: (r) => r.targetSlotNumber ?? 99,
  indexer: (r) => r.indexer,
  size: (r) => r.size,
  age: (r) => (r.publishDate ? new Date(r.publishDate).getTime() : 0),
  peers: (r) => r.seeders,
}

export function compareReleases(a: TorrentInfo, b: TorrentInfo, column: SortColumn): number {
  const getter = COMPARATORS[column]
  const aVal = getter(a)
  const bVal = getter(b)

  if (typeof aVal === 'string' && typeof bVal === 'string') {
    return aVal.localeCompare(bVal)
  }
  return (aVal as number) - (bVal as number)
}

type MediaTypeInfo = {
  mediaType: 'movie' | 'episode' | 'season'
  isSeasonPack: boolean
  isCompleteSeries: boolean
}

type ResolveMediaTypeParams = {
  isMovie: boolean
  seriesId?: number
  season?: number
  episode?: number
}

export function resolveMediaType(params: ResolveMediaTypeParams): MediaTypeInfo {
  const { isMovie, seriesId, season, episode } = params
  if (isMovie) {
    return { mediaType: 'movie', isSeasonPack: false, isCompleteSeries: false }
  }
  if (seriesId && season !== undefined && episode === undefined) {
    return { mediaType: 'season', isSeasonPack: true, isCompleteSeries: false }
  }
  if (seriesId && season === undefined && episode === undefined) {
    return { mediaType: 'season', isSeasonPack: false, isCompleteSeries: true }
  }
  return { mediaType: 'episode', isSeasonPack: false, isCompleteSeries: false }
}

type BuildGrabParams = {
  release: TorrentInfo
  mediaTypeInfo: MediaTypeInfo
  mediaId: number | undefined
  seriesId: number | undefined
  season: number | undefined
}

export function buildGrabRequest(params: BuildGrabParams): GrabRequest {
  const { release, mediaTypeInfo, mediaId, seriesId, season } = params

  return {
    release: {
      guid: release.guid,
      title: release.title,
      downloadUrl: release.downloadUrl,
      indexerId: release.indexerId,
      indexer: release.indexer,
      protocol: release.protocol,
      size: release.size,
      tmdbId: release.tmdbId,
      tvdbId: release.tvdbId,
      imdbId: release.imdbId,
    },
    mediaType: mediaTypeInfo.mediaType,
    mediaId,
    seriesId,
    seasonNumber: season,
    isSeasonPack: mediaTypeInfo.isSeasonPack,
    isCompleteSeries: mediaTypeInfo.isCompleteSeries,
    targetSlotId: release.targetSlotId,
  }
}
