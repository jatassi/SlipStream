import { useQuery } from '@tanstack/react-query'
import { missingApi } from '@/api'

export const missingKeys = {
  all: ['missing'] as const,
  movies: () => [...missingKeys.all, 'movies'] as const,
  series: () => [...missingKeys.all, 'series'] as const,
  counts: () => [...missingKeys.all, 'counts'] as const,
}

export function useMissingMovies() {
  return useQuery({
    queryKey: missingKeys.movies(),
    queryFn: () => missingApi.getMovies(),
  })
}

export function useMissingSeries() {
  return useQuery({
    queryKey: missingKeys.series(),
    queryFn: () => missingApi.getSeries(),
  })
}

export function useMissingCounts() {
  return useQuery({
    queryKey: missingKeys.counts(),
    queryFn: () => missingApi.getCounts(),
  })
}
