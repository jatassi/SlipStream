export type DownloadClientType = 'qbittorrent' | 'transmission' | 'sabnzbd' | 'nzbget'

export type DownloadClient = {
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

export type CreateDownloadClientInput = {
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

export type UpdateDownloadClientInput = {
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

export type DownloadClientTestResult = {
  success: boolean
  message: string
}
