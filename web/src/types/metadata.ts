export interface MovieSearchResult {
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

export interface SeriesSearchResult {
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

export interface MetadataImages {
  posters: ImageInfo[]
  backdrops: ImageInfo[]
}

export interface ImageInfo {
  path: string
  width: number
  height: number
  voteAverage?: number
}

export interface Person {
  id: number
  name: string
  role?: string
  photoUrl?: string
}

export interface Credits {
  directors?: Person[]
  writers?: Person[]
  creators?: Person[]
  cast: Person[]
}

export interface ExternalRatings {
  imdbRating?: number
  imdbVotes?: number
  rottenTomatoes?: number
  rottenAudience?: number
  metacritic?: number
  awards?: string
}

export interface SeasonResult {
  seasonNumber: number
  name: string
  overview?: string
  posterUrl?: string
  airDate?: string
  episodes?: EpisodeResult[]
}

export interface EpisodeResult {
  episodeNumber: number
  seasonNumber: number
  title: string
  overview?: string
  airDate?: string
  runtime?: number
}

export interface ExtendedMovieResult extends MovieSearchResult {
  credits?: Credits
  contentRating?: string
  studio?: string
  ratings?: ExternalRatings
}

export interface ExtendedSeriesResult extends SeriesSearchResult {
  credits?: Credits
  contentRating?: string
  ratings?: ExternalRatings
  seasons?: SeasonResult[]
}
