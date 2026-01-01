export interface Series {
  id: number
  title: string
  sortTitle: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId?: number
  qualityProfileId?: number
  monitored: boolean
  seasonFolder: boolean
  status: 'continuing' | 'ended' | 'upcoming'
  addedAt: string
  updatedAt?: string
  episodeCount: number
  episodeFileCount: number
  sizeOnDisk?: number
  seasons?: Season[]
}

export interface Season {
  id: number
  seriesId: number
  seasonNumber: number
  monitored: boolean
  episodeCount: number
  episodeFileCount: number
  sizeOnDisk?: number
  episodes?: Episode[]
}

export interface Episode {
  id: number
  seriesId: number
  seasonNumber: number
  episodeNumber: number
  title: string
  overview?: string
  airDate?: string
  monitored: boolean
  hasFile: boolean
  episodeFile?: EpisodeFile
}

export interface EpisodeFile {
  id: number
  episodeId: number
  path: string
  size: number
  quality?: string
  videoCodec?: string
  audioCodec?: string
  resolution?: string
  createdAt: string
}

export interface CreateSeriesInput {
  title: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId: number
  qualityProfileId: number
  monitored: boolean
  seasonFolder: boolean
  seasons?: SeasonInput[]
}

export interface AddSeriesInput extends CreateSeriesInput {
  posterUrl?: string
  backdropUrl?: string
}

export interface SeasonInput {
  seasonNumber: number
  monitored: boolean
  episodes?: EpisodeInput[]
}

export interface EpisodeInput {
  episodeNumber: number
  title: string
  overview?: string
  airDate?: string
  monitored: boolean
}

export interface UpdateSeriesInput {
  title?: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  path?: string
  rootFolderId?: number
  qualityProfileId?: number
  monitored?: boolean
  seasonFolder?: boolean
  status?: string
}

export interface UpdateEpisodeInput {
  title?: string
  overview?: string
  airDate?: string
  monitored?: boolean
}

export interface ListSeriesOptions {
  search?: string
  monitored?: boolean
  rootFolderId?: number
  page?: number
  pageSize?: number
}
