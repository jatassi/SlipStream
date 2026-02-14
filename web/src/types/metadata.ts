export type MovieSearchResult = {
  id: number
  tmdbId: number
  imdbId?: string
  title: string
  originalTitle?: string
  year?: number
  overview?: string
  posterUrl?: string
  backdropUrl?: string
  voteAverage?: number
  runtime?: number
  genres?: string[]
  releaseDate?: string
  studio?: string
}

export type SeriesSearchResult = {
  id: number
  tvdbId?: number
  tmdbId: number
  imdbId?: string
  title: string
  originalTitle?: string
  year?: number
  overview?: string
  posterUrl?: string
  backdropUrl?: string
  voteAverage?: number
  runtime?: number
  genres?: string[]
  status?: string
  network?: string
  networkLogoUrl?: string
  firstAirDate?: string
}

export type MetadataImages = {
  posters: ImageInfo[]
  backdrops: ImageInfo[]
}

export type ImageInfo = {
  path: string
  width: number
  height: number
  voteAverage?: number
}

export type Person = {
  id: number
  name: string
  role?: string
  photoUrl?: string
}

export type Credits = {
  directors?: Person[]
  writers?: Person[]
  creators?: Person[]
  cast: Person[]
}

export type ExternalRatings = {
  imdbRating?: number
  imdbVotes?: number
  rottenTomatoes?: number
  rottenAudience?: number
  metacritic?: number
  awards?: string
}

export type SeasonResult = {
  seasonNumber: number
  name: string
  overview?: string
  posterUrl?: string
  airDate?: string
  episodes?: EpisodeResult[]
}

export type EpisodeResult = {
  episodeNumber: number
  seasonNumber: number
  title: string
  overview?: string
  airDate?: string
  runtime?: number
  imdbRating?: number
}

export type ExtendedMovieResult = {
  credits?: Credits
  contentRating?: string
  studio?: string
  ratings?: ExternalRatings
} & MovieSearchResult

export type ExtendedSeriesResult = {
  credits?: Credits
  contentRating?: string
  ratings?: ExternalRatings
  seasons?: SeasonResult[]
} & SeriesSearchResult
