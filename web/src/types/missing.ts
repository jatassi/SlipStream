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

export interface UpgradableMovie {
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

export interface UpgradableEpisode {
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

export interface UpgradableSeason {
  seasonNumber: number
  upgradableEpisodes: UpgradableEpisode[]
}

export interface UpgradableSeries {
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

export interface UpgradableCounts {
  movies: number
  episodes: number
}
