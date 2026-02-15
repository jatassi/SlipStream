import { useQuery } from '@tanstack/react-query'

import { missingApi } from '@/api'

export const missingKeys = {
  all: ['missing'] as const,
  movies: () => [...missingKeys.all, 'movies'] as const,
  series: () => [...missingKeys.all, 'series'] as const,
  counts: () => [...missingKeys.all, 'counts'] as const,
  upgradableMovies: () => [...missingKeys.all, 'upgradable-movies'] as const,
  upgradableSeries: () => [...missingKeys.all, 'upgradable-series'] as const,
  upgradableCounts: () => [...missingKeys.all, 'upgradable-counts'] as const,
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

export function useUpgradableMovies() {
  return useQuery({
    queryKey: missingKeys.upgradableMovies(),
    queryFn: () => missingApi.getUpgradableMovies(),
  })
}

export function useUpgradableSeries() {
  return useQuery({
    queryKey: missingKeys.upgradableSeries(),
    queryFn: () => missingApi.getUpgradableSeries(),
  })
}

export function useUpgradableCounts() {
  return useQuery({
    queryKey: missingKeys.upgradableCounts(),
    queryFn: () => missingApi.getUpgradableCounts(),
  })
}
