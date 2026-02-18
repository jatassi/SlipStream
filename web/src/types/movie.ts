export type Movie = {
  id: number
  title: string
  sortTitle: string
  year?: number
  tmdbId?: number
  tvdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  studio?: string
  contentRating?: string
  path?: string
  rootFolderId?: number
  qualityProfileId: number
  monitored: boolean
  status: 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
  statusMessage?: string | null
  activeDownloadId?: string | null
  addedAt: string
  updatedAt?: string
  sizeOnDisk?: number
  movieFiles?: MovieFile[]
  releaseDate?: string
  physicalReleaseDate?: string
  theatricalReleaseDate?: string
  addedBy?: number
  addedByUsername?: string
}

export type MovieFile = {
  id: number
  movieId: number
  path: string
  size: number
  quality?: string
  videoCodec?: string
  audioCodec?: string
  audioChannels?: string
  dynamicRange?: string
  resolution?: string
  createdAt: string
  slotId?: number
}

export type CreateMovieInput = {
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

export type AddMovieInput = {
  posterUrl?: string
  backdropUrl?: string
  searchOnAdd?: boolean
} & CreateMovieInput

export type UpdateMovieInput = {
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

export type ListMoviesOptions = {
  search?: string
  monitored?: boolean
  rootFolderId?: number
}
