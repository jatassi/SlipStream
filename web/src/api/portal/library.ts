import type { PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { portalFetch } from './client'

export const portalLibraryApi = {
  getMovies: () => portalFetch<PortalMovieSearchResult[]>('/library/movies'),
  getSeries: () => portalFetch<PortalSeriesSearchResult[]>('/library/series'),
}
