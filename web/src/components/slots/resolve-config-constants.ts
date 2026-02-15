import type { AttributeMode } from '@/types'

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

export const HDR_FORMATS = ['DV', 'HDR10+', 'HDR10', 'HDR', 'HLG']

export type AttributeSettingsField =
  | 'hdrSettings'
  | 'videoCodecSettings'
  | 'audioCodecSettings'
  | 'audioChannelSettings'
