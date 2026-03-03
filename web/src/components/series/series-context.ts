import { createContext, useContext } from 'react'

export type SeriesInfo = {
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

export const SeriesContext = createContext<SeriesInfo | null>(null)

export function useSeriesInfo(): SeriesInfo {
  const ctx = useContext(SeriesContext)
  if (!ctx) {
    throw new Error('useSeriesInfo must be used within SeriesContext.Provider')
  }
  return ctx
}
