export type DownloadClientType =
  | 'transmission'
  | 'qbittorrent'
  | 'deluge'
  | 'rtorrent'
  | 'vuze'
  | 'flood'
  | 'aria2'
  | 'utorrent'
  | 'hadouken'
  | 'downloadstation'
  | 'freeboxdownload'
  | 'rqbit'
  | 'tribler'

export type DownloadClient = {
  id: number
  name: string
  type: DownloadClientType
  host: string
  port: number
  username?: string
  password?: string
  useSsl: boolean
  apiKey?: string
  category?: string
  urlBase?: string
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
  apiKey?: string
  category?: string
  urlBase?: string
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
  apiKey?: string
  category?: string
  urlBase?: string
  priority?: number
  enabled?: boolean
}

export type DownloadClientTestResult = {
  success: boolean
  message: string
}
