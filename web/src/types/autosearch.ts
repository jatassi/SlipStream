import type { TorrentInfo } from './search'

export type AutoSearchMediaType = 'movie' | 'episode' | 'season' | 'series'

export type AutoSearchSource = 'manual' | 'scheduled' | 'add'

export type AutoSearchResult = {
  found: boolean
  downloaded: boolean
  release?: TorrentInfo
  error?: string
  upgraded: boolean
  clientName?: string
  downloadId?: string
}

export type SlotSearchResult = {
  slotId: number
  slotNumber: number
  slotName: string
  isSlotUpgrade: boolean
} & AutoSearchResult

export type BatchAutoSearchResult = {
  totalSearched: number
  found: number
  downloaded: number
  failed: number
  results?: AutoSearchResult[]
}

export type AutoSearchStatus = {
  mediaType: AutoSearchMediaType
  mediaId: number
  searching: boolean
  inQueue: boolean
  lastSearch?: string
}

export type AutoSearchSettings = {
  enabled: boolean
  intervalHours: number
  backoffThreshold: number
}
