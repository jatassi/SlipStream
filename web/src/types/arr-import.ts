export type SourceType = 'radarr' | 'sonarr'

export type DBCandidate = {
  path: string
  exists: boolean
}

export type DetectDBResponse = {
  candidates: DBCandidate[]
  found: string
}

export type ConnectionConfig = {
  sourceType: SourceType
  dbPath?: string
  url?: string
  apiKey?: string
}

export type SourceRootFolder = {
  id: number
  path: string
}

export type SourceQualityProfile = {
  id: number
  name: string
  inUse: boolean
}

export type ImportMappings = {
  rootFolderMapping: Record<string, number>
  qualityProfileMapping: Record<number, number>
}

export type MoviePreview = {
  title: string
  year: number
  tmdbId: number
  hasFile: boolean
  quality: string
  status: 'new' | 'duplicate' | 'skip'
  skipReason?: string
}

export type SeriesPreview = {
  title: string
  year: number
  tvdbId: number
  episodeCount: number
  fileCount: number
  status: 'new' | 'duplicate' | 'skip'
  skipReason?: string
}

export type ImportSummary = {
  totalMovies: number
  totalSeries: number
  totalEpisodes: number
  totalFiles: number
  newMovies: number
  newSeries: number
  duplicateMovies: number
  duplicateSeries: number
  skippedMovies: number
  skippedSeries: number
}

export type ImportPreview = {
  movies: MoviePreview[]
  series: SeriesPreview[]
  summary: ImportSummary
}

export type ImportReport = {
  moviesCreated: number
  moviesSkipped: number
  moviesErrored: number
  seriesCreated: number
  seriesSkipped: number
  seriesErrored: number
  filesImported: number
  errors: string[]
}
