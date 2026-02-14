export type RssSyncSettings = {
  enabled: boolean
  intervalMin: number
}

export type RssSyncStatus = {
  running: boolean
  lastRun?: string
  totalReleases: number
  matched: number
  grabbed: number
  elapsed: number
  error?: string
}
