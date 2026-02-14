export type StatusCounts = {
  unreleased: number
  missing: number
  downloading: number
  failed: number
  upgradable: number
  available: number
  total: number
}

export type Series = {
  id: number
  title: string
  sortTitle: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  network?: string
  networkLogoUrl?: string
  path?: string
  rootFolderId?: number
  qualityProfileId: number
  monitored: boolean
  seasonFolder: boolean
  productionStatus: 'continuing' | 'ended' | 'upcoming'
  statusCounts: StatusCounts
  firstAired?: string
  lastAired?: string
  nextAiring?: string
  addedAt: string
  updatedAt?: string
  sizeOnDisk?: number
  seasons?: Season[]
  addedBy?: number
  addedByUsername?: string
}

export type Season = {
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

export type Episode = {
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

export type EpisodeFile = {
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

export type CreateSeriesInput = {
  title: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  overview?: string
  runtime?: number
  network?: string
  networkLogoUrl?: string
  path?: string
  rootFolderId: number
  qualityProfileId: number
  monitored: boolean
  seasonFolder: boolean
  seasons?: SeasonInput[]
}

export type SeriesSearchOnAdd = 'no' | 'first_episode' | 'first_season' | 'latest_season' | 'all'
export type SeriesMonitorOnAdd = 'none' | 'first_season' | 'latest_season' | 'future' | 'all'

export type AddSeriesInput = {
  posterUrl?: string
  backdropUrl?: string
  searchOnAdd?: SeriesSearchOnAdd
  monitorOnAdd?: SeriesMonitorOnAdd
  includeSpecials?: boolean
} & CreateSeriesInput

export type SeasonInput = {
  seasonNumber: number
  monitored: boolean
  episodes?: EpisodeInput[]
}

export type EpisodeInput = {
  episodeNumber: number
  title: string
  overview?: string
  airDate?: string
  monitored: boolean
}

export type UpdateSeriesInput = {
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

export type UpdateEpisodeInput = {
  title?: string
  overview?: string
  airDate?: string
  monitored?: boolean
}

export type ListSeriesOptions = {
  search?: string
  monitored?: boolean
  rootFolderId?: number
  page?: number
  pageSize?: number
}

// Bulk monitoring types
export type MonitorType = 'all' | 'none' | 'future' | 'first_season' | 'latest_season'

export type BulkMonitorInput = {
  monitorType: MonitorType
  includeSpecials: boolean
}

export type BulkEpisodeMonitorInput = {
  episodeIds: number[]
  monitored: boolean
}

export type MonitoringStats = {
  totalSeasons: number
  monitoredSeasons: number
  totalEpisodes: number
  monitoredEpisodes: number
}
