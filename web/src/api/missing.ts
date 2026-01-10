import { apiFetch } from './client'
import type { MissingMovie, MissingSeries, MissingCounts } from '@/types/missing'

export const missingApi = {
  getMovies: () => apiFetch<MissingMovie[]>('/missing/movies'),
  getSeries: () => apiFetch<MissingSeries[]>('/missing/series'),
  getCounts: () => apiFetch<MissingCounts>('/missing/counts'),
}
