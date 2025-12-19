export type HistoryEventType = 'grabbed' | 'imported' | 'deleted' | 'failed' | 'renamed'

export interface HistoryEntry {
  id: number
  eventType: HistoryEventType
  mediaType: 'movie' | 'series'
  mediaId: number
  source?: string
  quality?: string
  data?: Record<string, unknown>
  createdAt: string
  // Expanded media info
  mediaTitle?: string
}

export interface ListHistoryOptions {
  eventType?: HistoryEventType
  mediaType?: 'movie' | 'series'
  mediaId?: number
  page?: number
  pageSize?: number
}

export interface HistoryResponse {
  items: HistoryEntry[]
  page: number
  pageSize: number
  totalCount: number
  totalPages: number
}
