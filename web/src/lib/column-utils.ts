import type { ColumnDef } from './column-types'

export function getDefaultVisibleColumns<T>(columns: ColumnDef<T>[]): string[] {
  return columns.filter((c) => c.defaultVisible).map((c) => c.id)
}

export const DEFAULT_SORT_DIRECTIONS: Record<string, 'asc' | 'desc'> = {
  title: 'asc',
  monitored: 'desc',
  qualityProfile: 'asc',
  releaseDate: 'desc',
  dateAdded: 'desc',
  nextAirDate: 'desc',
  rootFolder: 'asc',
  sizeOnDisk: 'desc',
}
