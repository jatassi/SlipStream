export type DownloadClientType = 'qbittorrent' | 'transmission' | 'sabnzbd' | 'nzbget'

export interface DownloadClient {
  id: number
  name: string
  type: DownloadClientType
  host: string
  port: number
  username?: string
  password?: string
  useSsl: boolean
  category?: string
  priority: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateDownloadClientInput {
  name: string
  type: DownloadClientType
  host: string
  port: number
  username?: string
  password?: string
  useSsl?: boolean
  category?: string
  priority?: number
  enabled?: boolean
}

export interface UpdateDownloadClientInput {
  name?: string
  type?: DownloadClientType
  host?: string
  port?: number
  username?: string
  password?: string
  useSsl?: boolean
  category?: string
  priority?: number
  enabled?: boolean
}

export interface DownloadClientTestResult {
  success: boolean
  message: string
}
