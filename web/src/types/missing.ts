export interface MissingMovie {
  id: number
  title: string
  year?: number
  tmdbId?: number
  imdbId?: string
  releaseDate?: string
  physicalReleaseDate?: string
  path?: string
  qualityProfileId: number
}

export interface MissingEpisode {
  id: number
  seriesId: number
  seasonNumber: number
  episodeNumber: number
  title: string
  airDate?: string
  seriesTitle: string
  seriesTvdbId?: number
  seriesTmdbId?: number
  seriesImdbId?: string
  seriesYear?: number
}

export interface MissingSeason {
  seasonNumber: number
  missingEpisodes: MissingEpisode[]
}

export interface MissingSeries {
  id: number
  title: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  qualityProfileId: number
  missingCount: number
  missingSeasons: MissingSeason[]
}

export interface MissingCounts {
  movies: number
  episodes: number
}
