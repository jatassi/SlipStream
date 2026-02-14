import type {
  MissingCounts,
  MissingMovie,
  MissingSeries,
  UpgradableCounts,
  UpgradableMovie,
  UpgradableSeries,
} from '@/types/missing'

import { apiFetch } from './client'

export const missingApi = {
  getMovies: () => apiFetch<MissingMovie[]>('/missing/movies'),
  getSeries: () => apiFetch<MissingSeries[]>('/missing/series'),
  getCounts: () => apiFetch<MissingCounts>('/missing/counts'),

  getUpgradableMovies: () => apiFetch<UpgradableMovie[]>('/missing/upgradable/movies'),
  getUpgradableSeries: () => apiFetch<UpgradableSeries[]>('/missing/upgradable/series'),
  getUpgradableCounts: () => apiFetch<UpgradableCounts>('/missing/upgradable/counts'),
}
