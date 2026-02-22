import type { Privacy, Protocol } from './indexer'

// IndexerMode represents the active indexer management mode
export type IndexerMode = 'slipstream' | 'prowlarr'

// ProwlarrConfig holds Prowlarr connection and behavior configuration
export type ProwlarrConfig = {
  id: number
  enabled: boolean
  url: string
  apiKey: string
  movieCategories: number[]
  tvCategories: number[]
  timeout: number
  skipSslVerify: boolean
  capabilities?: ProwlarrCapabilities
  capabilitiesUpdatedAt?: string
  createdAt: string
  updatedAt: string
}

// ProwlarrConfigInput is the input for updating Prowlarr configuration
export type ProwlarrConfigInput = {
  enabled: boolean
  url: string
  apiKey: string
  movieCategories?: number[]
  tvCategories?: number[]
  timeout?: number
  skipSslVerify?: boolean
}

// ProwlarrTestInput is the input for testing Prowlarr connection
export type ProwlarrTestInput = {
  url: string
  apiKey: string
  timeout?: number
  skipSslVerify?: boolean
}

// ProwlarrTestResult is the result of testing Prowlarr connection
export type ProwlarrTestResult = {
  success: boolean
  message?: string
}

// ProwlarrIndexer represents an indexer configured in Prowlarr
export type ProwlarrIndexer = {
  id: number
  name: string
  protocol: Protocol
  privacy?: Privacy
  priority: number
  enable: boolean
  status: ProwlarrIndexerStatus
  capabilities?: ProwlarrIndexerCapabilities
}

// ProwlarrIndexerStatus represents the health status of a Prowlarr indexer
export type ProwlarrIndexerStatus = 0 | 1 | 2 | 3 // Healthy, Warning, Disabled, Failed

// ProwlarrIndexerStatusLabels maps status codes to display labels
export const ProwlarrIndexerStatusLabels: Record<ProwlarrIndexerStatus, string> = {
  0: 'Healthy',
  1: 'Warning',
  2: 'Disabled',
  3: 'Failed',
}

// ProwlarrIndexerCapabilities represents the capabilities of a Prowlarr indexer
export type ProwlarrIndexerCapabilities = {
  supportsSearch: boolean
  supportsTvSearch: boolean
  supportsMovieSearch: boolean
  categories?: number[]
}

// ProwlarrCapabilities represents the aggregated capabilities of Prowlarr
export type ProwlarrCapabilities = {
  supportsMovies: boolean
  supportsTv: boolean
  supportsSearch: boolean
  supportsRss: boolean
  categories: ProwlarrCategory[]
  indexerCount: number
  enabledIndexerCount: number
}

// ProwlarrCategory represents a Newznab category from Prowlarr
export type ProwlarrCategory = {
  id: number
  name: string
  description?: string
  subCategories?: ProwlarrCategory[]
}

// ProwlarrConnectionStatus represents the connection status to Prowlarr
export type ProwlarrConnectionStatus = {
  connected: boolean
  version?: string
  lastChecked?: string
  error?: string
}

// ModeInfo provides detailed information about the current indexer mode state
export type ModeInfo = {
  effectiveMode: IndexerMode
  configuredMode: IndexerMode
  devModeOverride: boolean
}

// SetModeInput is the input for setting the indexer mode
export type SetModeInput = {
  mode: IndexerMode
}

// RefreshResult is the result of refreshing Prowlarr data
export type RefreshResult = {
  indexers: ProwlarrIndexer[]
  refreshed: boolean
}

// Default Newznab movie category IDs
export const DEFAULT_MOVIE_CATEGORIES = [2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060]

// Default Newznab TV category IDs
export const DEFAULT_TV_CATEGORIES = [5000, 5010, 5020, 5030, 5040, 5045, 5050, 5060, 5070, 5080]

// Helper to get category name from ID
export function getCategoryName(categoryId: number): string {
  const allCategories: Record<number, string> = {
    2000: 'Movies',
    2010: 'Movies/Foreign',
    2020: 'Movies/Other',
    2030: 'Movies/SD',
    2040: 'Movies/HD',
    2045: 'Movies/UHD',
    2050: 'Movies/BluRay',
    2060: 'Movies/3D',
    5000: 'TV',
    5010: 'TV/WEB-DL',
    5020: 'TV/Foreign',
    5030: 'TV/SD',
    5040: 'TV/HD',
    5045: 'TV/UHD',
    5050: 'TV/Other',
    5060: 'TV/Sport',
    5070: 'TV/Anime',
    5080: 'TV/Documentary',
  }
  return allCategories[categoryId] ?? `Category ${categoryId}`
}

// ContentType represents what content types an indexer should be used for
export type ContentType = 'movies' | 'series' | 'both'

// ProwlarrIndexerSettings holds per-indexer configuration stored in SlipStream
export type ProwlarrIndexerSettings = {
  prowlarrIndexerId: number
  priority: number
  contentType: ContentType
  movieCategories?: number[]
  tvCategories?: number[]
  successCount: number
  failureCount: number
  lastFailureAt?: string
  lastFailureReason?: string
  createdAt: string
  updatedAt: string
}

// ProwlarrIndexerSettingsInput is used for creating/updating indexer settings
export type ProwlarrIndexerSettingsInput = {
  priority: number
  contentType: ContentType
  movieCategories?: number[]
  tvCategories?: number[]
}

// ProwlarrIndexerWithSettings combines Prowlarr indexer data with SlipStream settings
export type ProwlarrIndexerWithSettings = {
  settings?: ProwlarrIndexerSettings
} & ProwlarrIndexer

// ContentType labels for display
export const ContentTypeLabels: Record<ContentType, string> = {
  movies: 'Movies Only',
  series: 'Series Only',
  both: 'Both',
}
