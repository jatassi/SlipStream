export interface Movie {
  id: number
  title: string
  sortTitle: string
  year?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId?: number
  qualityProfileId?: number
  monitored: boolean
  status: 'missing' | 'downloading' | 'available'
  addedAt: string
  updatedAt?: string
  hasFile: boolean
  sizeOnDisk?: number
  movieFiles?: MovieFile[]
}

export interface MovieFile {
  id: number
  movieId: number
  path: string
  size: number
  quality?: string
  videoCodec?: string
  audioCodec?: string
  resolution?: string
  createdAt: string
}

export interface CreateMovieInput {
  title: string
  year?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId: number
  qualityProfileId: number
  monitored: boolean
}

export interface UpdateMovieInput {
  title?: string
  year?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId?: number
  qualityProfileId?: number
  monitored?: boolean
}

export interface ListMoviesOptions {
  search?: string
  monitored?: boolean
  rootFolderId?: number
  page?: number
  pageSize?: number
}
