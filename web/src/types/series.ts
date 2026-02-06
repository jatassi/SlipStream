export interface StatusCounts {
  unreleased: number
  missing: number
  downloading: number
  failed: number
  upgradable: number
  available: number
  total: number
}

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
  qualityProfileId: number
  monitored: boolean
  seasonFolder: boolean
  productionStatus: 'continuing' | 'ended' | 'upcoming'
  statusCounts: StatusCounts
  addedAt: string
  updatedAt?: string
  sizeOnDisk?: number
  seasons?: Season[]
}

export interface Season {
  id: number
  seriesId: number
  seasonNumber: number
  monitored: boolean
  overview?: string
  posterUrl?: string
  statusCounts: StatusCounts
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
  status: 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
  statusMessage?: string | null
  activeDownloadId?: string | null
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
  slotId?: number
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

export type SeriesSearchOnAdd = 'no' | 'first_episode' | 'first_season' | 'latest_season' | 'all'
export type SeriesMonitorOnAdd = 'none' | 'first_season' | 'latest_season' | 'future' | 'all'

export interface AddSeriesInput extends CreateSeriesInput {
  posterUrl?: string
  backdropUrl?: string
  searchOnAdd?: SeriesSearchOnAdd
  monitorOnAdd?: SeriesMonitorOnAdd
  includeSpecials?: boolean
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
  productionStatus?: string
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

// Bulk monitoring types
export type MonitorType = 'all' | 'none' | 'future' | 'first_season' | 'latest_season'

export interface BulkMonitorInput {
  monitorType: MonitorType
  includeSpecials: boolean
}

export interface BulkEpisodeMonitorInput {
  episodeIds: number[]
  monitored: boolean
}

export interface MonitoringStats {
  totalSeasons: number
  monitoredSeasons: number
  totalEpisodes: number
  monitoredEpisodes: number
}
