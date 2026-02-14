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

export type HistoryEntry = {
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

export type AutoSearchDownloadData = {
  releaseName?: string
  indexer?: string
  clientName?: string
  downloadId?: string
  source?: string
  isUpgrade?: boolean
  oldQuality?: string
  newQuality?: string
}

export type AutoSearchFailedData = {
  error?: string
  indexer?: string
  source?: string
}

export type ImportEventData = {
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

export type StatusChangedData = {
  from?: string
  to?: string
  reason?: string
}

export type FileRenamedData = {
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

export type ListHistoryOptions = {
  eventType?: string
  mediaType?: 'movie' | 'episode'
  mediaId?: number
  page?: number
  pageSize?: number
  before?: string
  after?: string
}

export type HistoryResponse = {
  items: HistoryEntry[]
  page: number
  pageSize: number
  totalCount: number
  totalPages: number
}
