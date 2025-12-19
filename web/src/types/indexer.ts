export interface Indexer {
  id: number
  name: string
  type: 'torznab' | 'newznab'
  url: string
  apiKey?: string
  categories?: string
  supportsMovies: boolean
  supportsTv: boolean
  priority: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateIndexerInput {
  name: string
  type: 'torznab' | 'newznab'
  url: string
  apiKey?: string
  categories?: string
  supportsMovies: boolean
  supportsTv: boolean
  priority?: number
  enabled?: boolean
}

export interface UpdateIndexerInput {
  name?: string
  type?: 'torznab' | 'newznab'
  url?: string
  apiKey?: string
  categories?: string
  supportsMovies?: boolean
  supportsTv?: boolean
  priority?: number
  enabled?: boolean
}

export interface IndexerTestResult {
  success: boolean
  message: string
}
