export interface MovieSearchResult {
  tmdbId: number
  imdbId?: string
  title: string
  originalTitle?: string
  year?: number
  overview?: string
  posterPath?: string
  backdropPath?: string
  voteAverage?: number
  runtime?: number
  genres?: string[]
  releaseDate?: string
}

export interface SeriesSearchResult {
  tvdbId?: number
  tmdbId: number
  imdbId?: string
  title: string
  originalTitle?: string
  year?: number
  overview?: string
  posterPath?: string
  backdropPath?: string
  voteAverage?: number
  runtime?: number
  genres?: string[]
  status?: string
  network?: string
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
