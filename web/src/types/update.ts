export type UpdateState =
  | 'idle'
  | 'checking'
  | 'up-to-date'
  | 'update-available'
  | 'error'
  | 'downloading'
  | 'installing'
  | 'restarting'
  | 'complete'
  | 'failed'

export type UpdateReleaseInfo = {
  version: string
  tagName: string
  releaseDate: string
  releaseNotes: string
  downloadUrl: string
  assetName: string
  assetSize: number
  publishedAt: string
}

export type UpdateStatus = {
  state: UpdateState
  currentVersion: string
  latestRelease?: UpdateReleaseInfo
  progress: number
  downloadedMB?: number
  totalMB?: number
  error?: string
  lastChecked?: string
}

export type UpdateSettings = {
  autoInstall: boolean
}
