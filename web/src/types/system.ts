export type SystemStatus = {
  version: string
  startTime: string
  uptime: number
  databaseSize: number
  movieCount: number
  seriesCount: number
  episodeCount: number
  queueCount: number
  developerMode: boolean
  portalEnabled: boolean
  mediainfoAvailable: boolean
  actualPort?: number
  configuredPort?: number
  tmdb?: {
    disableSearchOrdering: boolean
  }
}

export type HealthCheck = {
  status: 'healthy' | 'degraded' | 'unhealthy'
  checks: HealthCheckItem[]
}

export type HealthCheckItem = {
  name: string
  status: 'healthy' | 'degraded' | 'unhealthy'
  message?: string
}

export type Settings = {
  serverPort: number
  logLevel: string
  authEnabled: boolean
  apiKey: string
  logPath: string
  logMaxSizeMB: number
  logMaxBackups: number
  logMaxAgeDays: number
  logCompress: boolean
  externalAccessEnabled: boolean
}

export type UpdateSettingsInput = {
  serverPort?: number
  logLevel?: string
  authEnabled?: boolean
  password?: string
  logMaxSizeMB?: number
  logMaxBackups?: number
  logMaxAgeDays?: number
  logCompress?: boolean
  externalAccessEnabled?: boolean
}

export type FirewallStatus = {
  port: number
  isListening: boolean
  firewallAllows: boolean
  firewallName?: string
  firewallEnabled: boolean
  message?: string
  checkedAt: string
}
