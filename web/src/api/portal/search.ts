import type { EnrichedSeason, PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { buildQueryString, portalFetch } from './client'

export const portalSearchApi = {
  searchMovies: (query: string, init?: RequestInit) =>
    portalFetch<PortalMovieSearchResult[]>(`/search/movie${buildQueryString({ query })}`, init),

  searchSeries: (query: string, init?: RequestInit) =>
    portalFetch<PortalSeriesSearchResult[]>(`/search/series${buildQueryString({ query })}`, init),

  getSeriesSeasons: (tmdbId?: number, tvdbId?: number, init?: RequestInit) =>
    portalFetch<EnrichedSeason[]>(
      `/search/series/seasons${buildQueryString({ tmdbId, tvdbId })}`,
      init,
    ),
}
