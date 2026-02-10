export interface RssSyncSettings {
  enabled: boolean
  intervalMin: number
}

export interface RssSyncStatus {
  running: boolean
  lastRun?: string
  totalReleases: number
  matched: number
  grabbed: number
  elapsed: number
  error?: string
}
