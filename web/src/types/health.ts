// HealthStatus represents the health state of an item
export type HealthStatus = 'ok' | 'warning' | 'error'

// HealthCategory represents the category of health items
export type HealthCategory = 'downloadClients' | 'indexers' | 'rootFolders' | 'metadata' | 'storage'

// HealthItem represents a single health-tracked item
export interface HealthItem {
  id: string
  category: HealthCategory
  name: string
  status: HealthStatus
  message?: string
  timestamp?: string // ISO 8601, only present for warning/error
}

// CategorySummary provides counts for a health category
export interface CategorySummary {
  category: HealthCategory
  ok: number
  warning: number
  error: number
}

// HealthResponse contains all health items grouped by category
export interface HealthResponse {
  downloadClients: HealthItem[]
  indexers: HealthItem[]
  rootFolders: HealthItem[]
  metadata: HealthItem[]
  storage: HealthItem[]
}

// HealthSummary provides an overview of system health
export interface HealthSummary {
  categories: CategorySummary[]
  hasIssues: boolean
}

// TestCategoryResult represents the result of testing a category
export interface TestCategoryResult {
  category: HealthCategory
  results: TestItemResult[]
}

// TestItemResult represents the result of testing a single item
export interface TestItemResult {
  id: string
  success: boolean
  message: string
}

// HealthUpdatePayload is the WebSocket payload for health updates
export interface HealthUpdatePayload {
  category: HealthCategory
  id: string
  name: string
  status: HealthStatus
  message?: string
  timestamp?: string
}

// Helper to get display name for category
export function getCategoryDisplayName(category: HealthCategory): string {
  const names: Record<HealthCategory, string> = {
    downloadClients: 'Download Clients',
    indexers: 'Indexers',
    rootFolders: 'Root Folders',
    metadata: 'Metadata',
    storage: 'Storage',
  }
  return names[category]
}

// Helper to get settings path for category
export function getCategorySettingsPath(category: HealthCategory): string {
  const paths: Record<HealthCategory, string> = {
    downloadClients: '/settings/downloadclients',
    indexers: '/settings/indexers',
    rootFolders: '/settings/mediamanagement',
    metadata: '/settings/metadata',
    storage: '/settings/mediamanagement',
  }
  return paths[category]
}

// All categories in display order
export const ALL_HEALTH_CATEGORIES: HealthCategory[] = [
  'downloadClients',
  'indexers',
  'rootFolders',
  'metadata',
  'storage',
]
