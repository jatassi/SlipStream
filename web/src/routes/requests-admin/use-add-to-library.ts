import { useCallback } from 'react'

import { useAddMovie, useAddSeries, useQualityProfiles, useRequestSettings, useRootFolders } from '@/hooks'
import type { AddMovieInput, AddSeriesInput, Request, SeasonInput } from '@/types'

export function useAddToLibrary() {
  const { data: rootFolders } = useRootFolders()
  const { data: requestSettings } = useRequestSettings()
  const { data: qualityProfiles } = useQualityProfiles()
  const addMovieMutation = useAddMovie()
  const addSeriesMutation = useAddSeries()

  const getDefaultRootFolderId = useCallback(
    (mediaType: string) => {
      if (requestSettings?.defaultRootFolderId) {
        return requestSettings.defaultRootFolderId
      }
      const matchingFolder = rootFolders?.find(
        (f) =>
          (mediaType === 'movie' && f.mediaType === 'movie') ||
          (mediaType === 'series' && f.mediaType === 'tv'),
      )
      return matchingFolder?.id ?? rootFolders?.[0]?.id ?? 0
    },
    [requestSettings, rootFolders],
  )

  const getDefaultQualityProfileId = useCallback(() => {
    return qualityProfiles?.[0]?.id ?? 0
  }, [qualityProfiles])

  return async (request: Request) => {
    const rootFolderId = getDefaultRootFolderId(request.mediaType)
    const qualityProfileId = getDefaultQualityProfileId()

    if (!rootFolderId || !qualityProfileId) {
      throw new Error('Missing root folder or quality profile configuration')
    }

    const config = { rootFolderId, qualityProfileId }
    if (request.mediaType === 'movie') {
      return addMovie(request, config, addMovieMutation)
    }
    return addSeries(request, config, addSeriesMutation)
  }
}

type LibraryConfig = { rootFolderId: number; qualityProfileId: number }

async function addMovie(
  request: Request,
  config: LibraryConfig,
  mutation: ReturnType<typeof useAddMovie>,
) {
  const input: AddMovieInput = {
    title: request.title,
    year: request.year ?? undefined,
    tmdbId: request.tmdbId ?? undefined,
    rootFolderId: config.rootFolderId,
    qualityProfileId: config.qualityProfileId,
    monitored: true,
    posterUrl: request.posterUrl ?? undefined,
    searchOnAdd: false,
  }
  const movie = await mutation.mutateAsync(input)
  return {
    mediaId: movie.id,
    qualityProfileId: movie.qualityProfileId,
    tmdbId: movie.tmdbId,
    imdbId: movie.imdbId,
    year: movie.year,
  }
}

async function addSeries(
  request: Request,
  config: LibraryConfig,
  mutation: ReturnType<typeof useAddSeries>,
) {
  const seasons = buildSeasonInputs(request)
  const input: AddSeriesInput = {
    title: request.title,
    year: request.year ?? undefined,
    tmdbId: request.tmdbId ?? undefined,
    tvdbId: request.tvdbId ?? undefined,
    rootFolderId: config.rootFolderId,
    qualityProfileId: config.qualityProfileId,
    monitored: true,
    seasonFolder: true,
    posterUrl: request.posterUrl ?? undefined,
    searchOnAdd: 'no',
    monitorOnAdd: 'none',
    seasons: seasons.length > 0 ? seasons : undefined,
  }
  const series = await mutation.mutateAsync(input)
  return {
    mediaId: series.id,
    qualityProfileId: series.qualityProfileId,
    tvdbId: series.tvdbId,
    year: series.year,
  }
}

function buildSeasonInputs(request: Request): SeasonInput[] {
  const seasons: SeasonInput[] = []
  if (request.requestedSeasons.length > 0) {
    for (const seasonNum of request.requestedSeasons) {
      seasons.push({ seasonNumber: seasonNum, monitored: true })
    }
  }
  if (request.monitorFuture) {
    seasons.push({ seasonNumber: -1, monitored: true })
  }
  return seasons
}
