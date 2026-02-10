export type HistoryEventType =
  | 'grabbed'
  | 'imported'
  | 'deleted'
  | 'failed'
  | 'file_renamed'
  | 'autosearch_download'
  | 'autosearch_failed'
  | 'import_failed'
  | 'status_changed'

export interface HistoryEntry {
  id: number
  eventType: HistoryEventType
  mediaType: 'movie' | 'episode'
  mediaId: number
  source?: string
  quality?: string
  data?: HistoryEventData
  createdAt: string
  mediaTitle?: string
  mediaQualifier?: string
  seriesId?: number
  year?: number
}

export interface AutoSearchDownloadData {
  releaseName?: string
  indexer?: string
  clientName?: string
  downloadId?: string
  source?: string
  isUpgrade?: boolean
  oldQuality?: string
  newQuality?: string
}

export interface AutoSearchFailedData {
  error?: string
  indexer?: string
  source?: string
}

export interface ImportEventData {
  sourcePath?: string
  destinationPath?: string
  originalFilename?: string
  finalFilename?: string
  quality?: string
  source?: string
  codec?: string
  size?: number
  error?: string
  isUpgrade?: boolean
  previousFile?: string
  previousQuality?: string
  newQuality?: string
  clientName?: string
  linkMode?: string
}

export interface StatusChangedData {
  from?: string
  to?: string
  reason?: string
}

export interface FileRenamedData {
  source_path?: string
  destination_path?: string
  old_filename?: string
  new_filename?: string
}

export type HistoryEventData =
  | AutoSearchDownloadData
  | AutoSearchFailedData
  | ImportEventData
  | StatusChangedData
  | FileRenamedData
  | Record<string, unknown>

export interface ListHistoryOptions {
  eventType?: HistoryEventType
  mediaType?: 'movie' | 'episode'
  mediaId?: number
  page?: number
  pageSize?: number
  before?: string
  after?: string
}

export interface HistoryResponse {
  items: HistoryEntry[]
  page: number
  pageSize: number
  totalCount: number
  totalPages: number
}
