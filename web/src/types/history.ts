export type HistoryEventType =
  | 'grabbed'
  | 'imported'
  | 'deleted'
  | 'failed'
  | 'renamed'
  | 'autosearch_download'
  | 'autosearch_upgrade'
  | 'autosearch_failed'

export interface HistoryEntry {
  id: number
  eventType: HistoryEventType
  mediaType: 'movie' | 'episode'
  mediaId: number
  source?: string
  quality?: string
  data?: HistoryEventData
  createdAt: string
  // Expanded media info
  mediaTitle?: string
}

export interface AutoSearchDownloadData {
  releaseName?: string
  indexer?: string
  clientName?: string
  downloadId?: string
  source?: string
}

export interface AutoSearchUpgradeData {
  releaseName?: string
  indexer?: string
  clientName?: string
  downloadId?: string
  oldQuality?: string
  newQuality?: string
  source?: string
}

export interface AutoSearchFailedData {
  error?: string
  indexer?: string
  source?: string
}

export type HistoryEventData =
  | AutoSearchDownloadData
  | AutoSearchUpgradeData
  | AutoSearchFailedData
  | Record<string, unknown>

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
