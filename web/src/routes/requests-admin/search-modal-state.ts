export type SearchModalState = {
  open: boolean
  mediaType: 'movie' | 'series'
  mediaId: number
  mediaTitle: string
  tmdbId?: number
  imdbId?: string
  tvdbId?: number
  qualityProfileId: number
  year?: number
  season?: number
  pendingSeasons?: number[]
}
