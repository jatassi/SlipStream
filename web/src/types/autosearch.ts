import type { TorrentInfo } from './search'

export type AutoSearchMediaType = 'movie' | 'episode' | 'season' | 'series'

export type AutoSearchSource = 'manual' | 'scheduled' | 'add'

export interface AutoSearchResult {
  found: boolean
  downloaded: boolean
  release?: TorrentInfo
  error?: string
  upgraded: boolean
  clientName?: string
  downloadId?: string
}

export interface SlotSearchResult extends AutoSearchResult {
  slotId: number
  slotNumber: number
  slotName: string
  isSlotUpgrade: boolean
}

export interface BatchAutoSearchResult {
  totalSearched: number
  found: number
  downloaded: number
  failed: number
  results?: AutoSearchResult[]
}

export interface AutoSearchStatus {
  mediaType: AutoSearchMediaType
  mediaId: number
  searching: boolean
  inQueue: boolean
  lastSearch?: string
}

export interface AutoSearchSettings {
  enabled: boolean
  intervalHours: number
  backoffThreshold: number
}
