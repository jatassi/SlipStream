export type MissingMovie = {
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

export type MissingEpisode = {
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

export type MissingSeason = {
  seasonNumber: number
  missingEpisodes: MissingEpisode[]
}

export type MissingSeries = {
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

export type MissingCounts = {
  movies: number
  episodes: number
}

export type UpgradableMovie = {
  id: number
  title: string
  year?: number
  tmdbId?: number
  imdbId?: string
  releaseDate?: string
  physicalReleaseDate?: string
  path?: string
  qualityProfileId: number
  currentQualityId: number
}

export type UpgradableEpisode = {
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
  currentQualityId: number
}

export type UpgradableSeason = {
  seasonNumber: number
  upgradableEpisodes: UpgradableEpisode[]
}

export type UpgradableSeries = {
  id: number
  title: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  qualityProfileId: number
  upgradableCount: number
  upgradableSeasons: UpgradableSeason[]
}

export type UpgradableCounts = {
  movies: number
  episodes: number
}
