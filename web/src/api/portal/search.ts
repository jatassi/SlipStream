import type { EnrichedSeason, PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { buildQueryString, portalFetch } from './client'

export const portalSearchApi = {
  searchMovies: (query: string) =>
    portalFetch<PortalMovieSearchResult[]>(`/search/movie${buildQueryString({ query })}`),

  searchSeries: (query: string) =>
    portalFetch<PortalSeriesSearchResult[]>(`/search/series${buildQueryString({ query })}`),

  getSeriesSeasons: (tmdbId?: number, tvdbId?: number) =>
    portalFetch<EnrichedSeason[]>(`/search/series/seasons${buildQueryString({ tmdbId, tvdbId })}`),
}
