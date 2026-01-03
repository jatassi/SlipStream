// Protocol represents the download protocol
export type Protocol = 'torrent' | 'usenet'

// Privacy represents indexer privacy level
export type Privacy = 'public' | 'semi-private' | 'private'

// Indexer represents a configured indexer instance
export interface Indexer {
  id: number
  name: string
  definitionId: string
  categories: number[]
  protocol: Protocol
  privacy: Privacy
  supportsMovies: boolean
  supportsTv: boolean
  supportsSearch: boolean
  supportsRss: boolean
  priority: number
  enabled: boolean
  settings?: Record<string, string>
  createdAt?: string
  updatedAt?: string
}

// CreateIndexerInput is the input for creating a new indexer
export interface CreateIndexerInput {
  name: string
  definitionId: string
  settings?: Record<string, string>
  categories?: number[]
  supportsMovies: boolean
  supportsTv: boolean
  priority?: number
  enabled?: boolean
}

// UpdateIndexerInput is the input for updating an indexer
export interface UpdateIndexerInput {
  name?: string
  definitionId?: string
  settings?: Record<string, string>
  categories?: number[]
  supportsMovies?: boolean
  supportsTv?: boolean
  priority?: number
  enabled?: boolean
}

// TestConfigInput is the input for testing an indexer configuration
export interface TestConfigInput {
  definitionId: string
  settings?: Record<string, string>
}

// IndexerTestResult is the result of testing an indexer
export interface IndexerTestResult {
  success: boolean
  message: string
  capabilities?: IndexerCapabilities
}

// IndexerCapabilities describes what an indexer supports
export interface IndexerCapabilities {
  supportsMovies: boolean
  supportsTv: boolean
  supportsSearch: boolean
  supportsRss: boolean
  searchParams?: string[]
  tvSearchParams?: string[]
  movieSearchParams?: string[]
  categories?: CategoryMapping[]
  maxResultsPerSearch?: number
}

// CategoryMapping maps indexer categories to standard categories
export interface CategoryMapping {
  id: number
  name: string
  description?: string
}

// IndexerStatus represents the health status of an indexer
export interface IndexerStatus {
  indexerId: number
  indexerName: string
  status: 'healthy' | 'warning' | 'failing' | 'disabled'
  message: string
  failureCount?: number
  disabledTill?: string
  lastRssSync?: string
}

// DefinitionMetadata contains metadata about a Cardigann definition
export interface DefinitionMetadata {
  id: string
  name: string
  description?: string
  language?: string
  protocol: Protocol
  privacy: Privacy
  siteUrl?: string
}

// DefinitionSetting describes a configurable setting for a definition
export interface DefinitionSetting {
  name: string
  type: 'text' | 'password' | 'checkbox' | 'select' | 'info'
  label: string
  default?: string
  options?: Record<string, string>
}

// DefinitionFilters for searching definitions
export interface DefinitionFilters {
  protocol?: Protocol
  privacy?: Privacy
  language?: string
}

// Definition contains full details about a Cardigann definition
export interface Definition {
  id: string
  name: string
  description?: string
  language?: string
  protocol: Protocol
  privacy: Privacy
  siteUrl?: string
  settings?: DefinitionSetting[]
}
