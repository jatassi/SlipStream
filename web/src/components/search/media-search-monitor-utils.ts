import { toast } from 'sonner'

import type { MediaTarget } from '@/hooks/use-media-download-progress'
import type { AutoSearchResult, BatchAutoSearchResult } from '@/types'

import type { MediaSearchMonitorControlsProps, SearchModalExternalProps } from './media-search-monitor-types'

export function formatSingleResult(result: AutoSearchResult, title: string): void {
  if (result.error) {
    toast.error(`Search failed for "${title}"`, { description: result.error })
    return
  }
  if (!result.found) {
    toast.warning(`No releases found for "${title}"`)
    return
  }
  if (result.downloaded) {
    const message = result.upgraded ? 'Quality upgrade found' : 'Found and downloading'
    toast.success(`${message}: ${result.release?.title ?? title}`, {
      description: result.clientName ? `Sent to ${result.clientName}` : undefined,
    })
  } else {
    toast.info(`Release found but not downloaded: ${result.release?.title ?? title}`)
  }
}

export function formatBatchResult(result: BatchAutoSearchResult, title: string): void {
  if (result.downloaded > 0) {
    toast.success(`Found ${result.downloaded} releases for "${title}"`, {
      description: `Searched ${result.totalSearched} items`,
    })
  } else if (result.found > 0) {
    toast.info(`Found ${result.found} releases but none downloaded for "${title}"`)
  } else if (result.failed > 0) {
    toast.error(`Search failed for ${result.failed} items in "${title}"`)
  } else {
    toast.warning(`No releases found for "${title}"`)
  }
}

export function buildDownloadTarget(props: MediaSearchMonitorControlsProps): MediaTarget {
  switch (props.mediaType) {
    case 'movie': {
      return { mediaType: 'movie', movieId: props.movieId }
    }
    case 'series': {
      return { mediaType: 'series', seriesId: props.seriesId }
    }
    case 'season': {
      return { mediaType: 'season', seriesId: props.seriesId, seasonNumber: props.seasonNumber }
    }
    case 'episode': {
      return {
        mediaType: 'episode',
        episodeId: props.episodeId,
        seriesId: props.seriesId,
        seasonNumber: props.seasonNumber,
      }
    }
    case 'movie-slot': {
      return { mediaType: 'movie-slot', movieId: props.movieId, slotId: props.slotId }
    }
    case 'episode-slot': {
      return {
        mediaType: 'episode-slot',
        episodeId: props.episodeId,
        slotId: props.slotId,
        seriesId: props.seriesId,
        seasonNumber: props.seasonNumber,
      }
    }
  }
}

export function buildSearchModalProps(
  props: MediaSearchMonitorControlsProps,
  qualityProfileId: number,
): SearchModalExternalProps {
  const base = { qualityProfileId }

  switch (props.mediaType) {
    case 'movie': {
      return { ...base, movieId: props.movieId, movieTitle: props.title, tmdbId: props.tmdbId, imdbId: props.imdbId, year: props.year }
    }
    case 'series': {
      return { ...base, seriesId: props.seriesId, seriesTitle: props.title, tvdbId: props.tvdbId }
    }
    case 'season': {
      return { ...base, seriesId: props.seriesId, seriesTitle: props.seriesTitle, tvdbId: props.tvdbId, season: props.seasonNumber }
    }
    case 'episode': {
      return { ...base, seriesId: props.seriesId, seriesTitle: props.seriesTitle, tvdbId: props.tvdbId, season: props.seasonNumber, episode: props.episodeNumber }
    }
    case 'movie-slot': {
      return { ...base, movieId: props.movieId, movieTitle: props.title, tmdbId: props.tmdbId, imdbId: props.imdbId, year: props.year }
    }
    case 'episode-slot': {
      return { ...base, seriesId: props.seriesId, seriesTitle: props.seriesTitle, tvdbId: props.tvdbId, season: props.seasonNumber, episode: props.episodeNumber }
    }
  }
}
