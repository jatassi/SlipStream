export interface SystemStatus {
  version: string
  startTime: string
  uptime: number
  databaseSize: number
  movieCount: number
  seriesCount: number
  episodeCount: number
  queueCount: number
  developerMode: boolean
  tmdb?: {
    disableSearchOrdering: boolean
  }
}

export interface HealthCheck {
  status: 'healthy' | 'degraded' | 'unhealthy'
  checks: HealthCheckItem[]
}

export interface HealthCheckItem {
  name: string
  status: 'healthy' | 'degraded' | 'unhealthy'
  message?: string
}

export interface Settings {
  serverPort: number
  logLevel: string
  authEnabled: boolean
  apiKey: string
}

export interface UpdateSettingsInput {
  serverPort?: number
  logLevel?: string
  authEnabled?: boolean
  password?: string
}
