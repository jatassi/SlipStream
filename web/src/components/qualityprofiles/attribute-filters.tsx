import type { AttributeMode, AttributeOptions, AttributeSettings } from '@/types'

import { AttributeSettingsSection } from './attribute-settings-section'

type AttributeValidation = {
  hdr: string | null
  videoCodec: string | null
  audioCodec: string | null
  audioChannels: string | null
}

type AttributeFiltersProps = {
  hdrSettings: AttributeSettings
  videoCodecSettings: AttributeSettings
  audioCodecSettings: AttributeSettings
  audioChannelSettings: AttributeSettings
  hdrOptions: string[]
  disabledHdrItems: string[]
  attributeOptions: AttributeOptions | undefined
  attributeValidation: AttributeValidation
  onItemModeChange: (
    field: 'hdrSettings' | 'videoCodecSettings' | 'audioCodecSettings' | 'audioChannelSettings',
    value: string,
    mode: AttributeMode,
  ) => void
}

export function AttributeFilters({
  hdrSettings,
  videoCodecSettings,
  audioCodecSettings,
  audioChannelSettings,
  hdrOptions,
  disabledHdrItems,
  attributeOptions,
  attributeValidation,
  onItemModeChange,
}: AttributeFiltersProps) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium">Attribute Filters</h3>

      <AttributeSettingsSection
        label="HDR Format"
        settings={hdrSettings}
        options={hdrOptions}
        disabledItems={disabledHdrItems}
        warning={attributeValidation.hdr}
        onItemModeChange={(value, mode) => onItemModeChange('hdrSettings', value, mode)}
      />

      <AttributeSettingsSection
        label="Video Codec"
        settings={videoCodecSettings}
        options={attributeOptions?.videoCodecs ?? []}
        warning={attributeValidation.videoCodec}
        onItemModeChange={(value, mode) => onItemModeChange('videoCodecSettings', value, mode)}
      />

      <AttributeSettingsSection
        label="Audio Codec"
        settings={audioCodecSettings}
        options={attributeOptions?.audioCodecs ?? []}
        warning={attributeValidation.audioCodec}
        onItemModeChange={(value, mode) => onItemModeChange('audioCodecSettings', value, mode)}
      />

      <AttributeSettingsSection
        label="Audio Channels"
        settings={audioChannelSettings}
        options={attributeOptions?.audioChannels ?? []}
        warning={attributeValidation.audioChannels}
        onItemModeChange={(value, mode) => onItemModeChange('audioChannelSettings', value, mode)}
      />
    </div>
  )
}
