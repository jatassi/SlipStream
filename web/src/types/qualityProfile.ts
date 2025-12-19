export interface Quality {
  id: number
  name: string
  source: string
  resolution: number
  weight: number
}

export interface QualityItem {
  quality: Quality
  allowed: boolean
}

export interface QualityProfile {
  id: number
  name: string
  cutoff: number
  items: QualityItem[]
  createdAt: string
  updatedAt: string
}

export interface CreateQualityProfileInput {
  name: string
  cutoff: number
  items: QualityItem[]
}

export interface UpdateQualityProfileInput {
  name: string
  cutoff: number
  items: QualityItem[]
}

export const PREDEFINED_QUALITIES: Quality[] = [
  { id: 1, name: 'SDTV', source: 'tv', resolution: 480, weight: 1 },
  { id: 2, name: 'DVD', source: 'dvd', resolution: 480, weight: 2 },
  { id: 3, name: 'WEBRip-480p', source: 'webrip', resolution: 480, weight: 3 },
  { id: 4, name: 'HDTV-720p', source: 'tv', resolution: 720, weight: 4 },
  { id: 5, name: 'WEBRip-720p', source: 'webrip', resolution: 720, weight: 5 },
  { id: 6, name: 'WEBDL-720p', source: 'webdl', resolution: 720, weight: 6 },
  { id: 7, name: 'Bluray-720p', source: 'bluray', resolution: 720, weight: 7 },
  { id: 8, name: 'HDTV-1080p', source: 'tv', resolution: 1080, weight: 8 },
  { id: 9, name: 'WEBRip-1080p', source: 'webrip', resolution: 1080, weight: 9 },
  { id: 10, name: 'WEBDL-1080p', source: 'webdl', resolution: 1080, weight: 10 },
  { id: 11, name: 'Bluray-1080p', source: 'bluray', resolution: 1080, weight: 11 },
  { id: 12, name: 'Remux-1080p', source: 'remux', resolution: 1080, weight: 12 },
  { id: 13, name: 'HDTV-2160p', source: 'tv', resolution: 2160, weight: 13 },
  { id: 14, name: 'WEBRip-2160p', source: 'webrip', resolution: 2160, weight: 14 },
  { id: 15, name: 'WEBDL-2160p', source: 'webdl', resolution: 2160, weight: 15 },
  { id: 16, name: 'Bluray-2160p', source: 'bluray', resolution: 2160, weight: 16 },
  { id: 17, name: 'Remux-2160p', source: 'remux', resolution: 2160, weight: 17 },
]
