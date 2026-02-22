// HealthStatus represents the health state of an item
export type HealthStatus = 'ok' | 'warning' | 'error'

// HealthCategory represents the category of health items
export type HealthCategory =
  | 'downloadClients'
  | 'indexers'
  | 'prowlarr'
  | 'rootFolders'
  | 'metadata'
  | 'storage'
  | 'import'

// HealthItem represents a single health-tracked item
export type HealthItem = {
  id: string
  category: HealthCategory
  name: string
  status: HealthStatus
  message?: string
  timestamp?: string // ISO 8601, only present for warning/error
}

// CategorySummary provides counts for a health category
export type CategorySummary = {
  category: HealthCategory
  ok: number
  warning: number
  error: number
}

// HealthResponse contains all health items grouped by category
export type HealthResponse = {
  downloadClients: HealthItem[]
  indexers: HealthItem[]
  prowlarr: HealthItem[]
  rootFolders: HealthItem[]
  metadata: HealthItem[]
  storage: HealthItem[]
  import: HealthItem[]
}

// HealthSummary provides an overview of system health
export type HealthSummary = {
  categories: CategorySummary[]
  hasIssues: boolean
}

// TestCategoryResult represents the result of testing a category
export type TestCategoryResult = {
  category: HealthCategory
  results: TestItemResult[]
}

// TestItemResult represents the result of testing a single item
export type TestItemResult = {
  id: string
  success: boolean
  message: string
}

// Helper to get display name for category
export function getCategoryDisplayName(category: HealthCategory): string {
  const names: Record<HealthCategory, string> = {
    downloadClients: 'Download Clients',
    indexers: 'Indexers',
    prowlarr: 'Prowlarr',
    rootFolders: 'Root Folders',
    metadata: 'Metadata',
    storage: 'Storage',
    import: 'Import',
  }
  return names[category]
}

// Helper to get settings path for category
export function getCategorySettingsPath(category: HealthCategory): string {
  const paths: Record<HealthCategory, string> = {
    downloadClients: '/settings/downloadclients',
    indexers: '/settings/indexers',
    prowlarr: '/settings/indexers',
    rootFolders: '/settings/mediamanagement',
    metadata: '/settings/metadata',
    storage: '/settings/mediamanagement',
    import: '/settings/import',
  }
  return paths[category]
}
