import type { AttributeMode, CreateQualityProfileInput, QualityItem, UpgradeStrategy } from '@/types'
import { DEFAULT_ATTRIBUTE_SETTINGS, PREDEFINED_QUALITIES } from '@/types'

export const MODE_OPTIONS: { value: AttributeMode; label: string }[] = [
  { value: 'required', label: 'Required' },
  { value: 'preferred', label: 'Preferred' },
  { value: 'acceptable', label: 'Acceptable' },
  { value: 'notAllowed', label: 'Not Allowed' },
]

export const MODE_LABELS: Record<AttributeMode, string> = {
  required: 'Required',
  preferred: 'Preferred',
  acceptable: 'Acceptable',
  notAllowed: 'Not Allowed',
}

export const UPGRADE_STRATEGY_OPTIONS: {
  value: UpgradeStrategy
  label: string
  description: string
}[] = [
  {
    value: 'balanced',
    label: 'Balanced',
    description: 'Upgrade for better resolution or source type',
  },
  {
    value: 'aggressive',
    label: 'Aggressive',
    description: 'Upgrade for any higher quality weight',
  },
  {
    value: 'resolution_only',
    label: 'Resolution Only',
    description: 'Only upgrade for higher resolution',
  },
]

export const HDR_FORMATS = ['DV', 'HDR10+', 'HDR10', 'HDR', 'HLG']

export const RESOLUTIONS = [480, 720, 1080, 2160] as const

export const defaultItems: QualityItem[] = PREDEFINED_QUALITIES.map((q) => ({
  quality: q,
  allowed: q.weight >= 10,
}))

export const defaultFormData: CreateQualityProfileInput = {
  name: '',
  cutoff: 10,
  upgradesEnabled: true,
  upgradeStrategy: 'balanced',
  cutoffOverridesStrategy: false,
  allowAutoApprove: false,
  items: defaultItems,
  hdrSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  videoCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioCodecSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
  audioChannelSettings: { ...DEFAULT_ATTRIBUTE_SETTINGS },
}
