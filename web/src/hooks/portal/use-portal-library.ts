import { useQuery } from '@tanstack/react-query'

import { portalLibraryApi } from '@/api'

const portalLibraryKeys = {
  all: ['portalLibrary'] as const,
  movies: () => [...portalLibraryKeys.all, 'movies'] as const,
  series: () => [...portalLibraryKeys.all, 'series'] as const,
}

export function usePortalLibraryMovies() {
  return useQuery({
    queryKey: portalLibraryKeys.movies(),
    queryFn: () => portalLibraryApi.getMovies(),
    staleTime: 5 * 60 * 1000,
  })
}

export function usePortalLibrarySeries() {
  return useQuery({
    queryKey: portalLibraryKeys.series(),
    queryFn: () => portalLibraryApi.getSeries(),
    staleTime: 5 * 60 * 1000,
  })
}
