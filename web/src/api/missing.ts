import { apiFetch } from './client'
import type { MissingMovie, MissingSeries, MissingCounts, UpgradableMovie, UpgradableSeries, UpgradableCounts } from '@/types/missing'

export const missingApi = {
  getMovies: () => apiFetch<MissingMovie[]>('/missing/movies'),
  getSeries: () => apiFetch<MissingSeries[]>('/missing/series'),
  getCounts: () => apiFetch<MissingCounts>('/missing/counts'),

  getUpgradableMovies: () => apiFetch<UpgradableMovie[]>('/missing/upgradable/movies'),
  getUpgradableSeries: () => apiFetch<UpgradableSeries[]>('/missing/upgradable/series'),
  getUpgradableCounts: () => apiFetch<UpgradableCounts>('/missing/upgradable/counts'),
}
