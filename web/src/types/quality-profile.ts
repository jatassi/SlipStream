export type Quality = {
  id: number
  name: string
  source: string
  resolution: number
  weight: number
}

export type QualityItem = {
  quality: Quality
  allowed: boolean
}

export type AttributeMode = 'acceptable' | 'preferred' | 'required' | 'notAllowed'

export type AttributeSettings = {
  items: Record<string, AttributeMode> // value -> mode mapping (e.g., { "DV": "required", "HDR10": "preferred" })
}

export type UpgradeStrategy = 'aggressive' | 'balanced' | 'resolution_only'

export type QualityProfile = {
  id: number
  name: string
  cutoff: number
  upgradesEnabled: boolean
  upgradeStrategy: UpgradeStrategy
  cutoffOverridesStrategy: boolean
  allowAutoApprove: boolean
  items: QualityItem[]
  hdrSettings: AttributeSettings
  videoCodecSettings: AttributeSettings
  audioCodecSettings: AttributeSettings
  audioChannelSettings: AttributeSettings
  createdAt: string
  updatedAt: string
}

export type CreateQualityProfileInput = {
  name: string
  cutoff: number
  upgradesEnabled: boolean
  upgradeStrategy: UpgradeStrategy
  cutoffOverridesStrategy: boolean
  allowAutoApprove: boolean
  items: QualityItem[]
  hdrSettings: AttributeSettings
  videoCodecSettings: AttributeSettings
  audioCodecSettings: AttributeSettings
  audioChannelSettings: AttributeSettings
}

export type UpdateQualityProfileInput = {
  name: string
  cutoff: number
  upgradesEnabled: boolean
  upgradeStrategy: UpgradeStrategy
  cutoffOverridesStrategy: boolean
  allowAutoApprove: boolean
  items: QualityItem[]
  hdrSettings: AttributeSettings
  videoCodecSettings: AttributeSettings
  audioCodecSettings: AttributeSettings
  audioChannelSettings: AttributeSettings
}

export type AttributeOptions = {
  hdrFormats: string[]
  videoCodecs: string[]
  audioCodecs: string[]
  audioChannels: string[]
  modes: AttributeMode[]
}

export const DEFAULT_ATTRIBUTE_SETTINGS: AttributeSettings = {
  items: {},
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

// Exclusivity checking types (Req 3.1.1-3.1.4)
export type ExclusivityDetail = {
  profileAId: number
  profileAName: string
  profileBId: number
  profileBName: string
  areExclusive: boolean
  conflicts?: string[]
  overlaps?: string[]
  hints?: string[]
}

export type SlotExclusivityError = {
  slotA: number
  slotB: number
  slotAName: string
  slotBName: string
  profileAName: string
  profileBName: string
  overlappingAttr?: string
  reason: string
}

export type CheckExclusivityResponse = {
  valid: boolean
  errors?: SlotExclusivityError[]
  details?: ExclusivityDetail[]
}
