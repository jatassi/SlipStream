import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type {
  AttributeMode,
  AttributeOptions,
  CreateQualityProfileInput,
  QualityProfile,
  Slot,
} from '@/types'

import { CompactAttributeSection } from './compact-attribute-section'
import type { AttributeSettingsField } from './resolve-config-constants'

const RESOLUTIONS = [480, 720, 1080, 2160] as const

type ProfileEditorCardProps = {
  profile: QualityProfile
  formData: CreateQualityProfileInput
  slots: Slot[] | undefined
  hdrOptions: string[]
  attributeOptions: AttributeOptions | undefined
  conflictingAttributes: Set<string>
  onUpdateField: (field: keyof CreateQualityProfileInput, value: unknown) => void
  onUpdateItemMode: (field: AttributeSettingsField, value: string, mode: AttributeMode) => void
  onToggleQuality: (qualityId: number) => void
}

export function ProfileEditorCard({
  profile,
  formData,
  slots,
  hdrOptions,
  attributeOptions,
  conflictingAttributes,
  onUpdateField,
  onUpdateItemMode,
  onToggleQuality,
}: ProfileEditorCardProps) {
  const slot = slots?.find((s) => s.qualityProfileId === profile.id)
  const allowedQualities = formData.items.filter((i) => i.allowed)
  const cutoffOptions = allowedQualities.length > 0 ? allowedQualities : formData.items

  return (
    <div className="space-y-4 rounded-lg border p-4">
      <div className="flex items-center gap-2">
        <Badge variant="outline">{slot?.name ?? 'Unknown Slot'}</Badge>
        <span className="font-medium">{profile.name}</span>
      </div>

      <div className="space-y-2">
        <Label>Name</Label>
        <Input value={formData.name} onChange={(e) => onUpdateField('name', e.target.value)} />
      </div>

      <QualityChecklist items={formData.items} onToggle={onToggleQuality} />

      <CutoffSelect formData={formData} cutoffOptions={cutoffOptions} onUpdateField={onUpdateField} />

      <AttributeFiltersSection
        formData={formData}
        hdrOptions={hdrOptions}
        attributeOptions={attributeOptions}
        conflictingAttributes={conflictingAttributes}
        onUpdateItemMode={onUpdateItemMode}
      />
    </div>
  )
}

function QualityChecklist({
  items,
  onToggle,
}: {
  items: CreateQualityProfileInput['items']
  onToggle: (qualityId: number) => void
}) {
  return (
    <div className="space-y-2">
      <Label>Allowed Qualities</Label>
      <div className="bg-muted/30 max-h-40 divide-y overflow-y-auto rounded-lg border">
        {RESOLUTIONS.map((resolution) => {
          const resolutionItems = items.filter((item) => item.quality.resolution === resolution)
          if (resolutionItems.length === 0) {
            return null
          }
          return (
            <div key={resolution} className="p-2">
              <div className="text-muted-foreground mb-1 text-xs font-medium">
                {resolution === 480 ? 'SD' : `${resolution}p`}
              </div>
              <div className="flex flex-wrap gap-x-3 gap-y-1">
                {resolutionItems.map((item) => (
                  <label key={item.quality.id} className="flex cursor-pointer items-center gap-1.5">
                    <Checkbox
                      checked={item.allowed}
                      onCheckedChange={() => onToggle(item.quality.id)}
                    />
                    <span className="text-xs">{item.quality.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function CutoffSelect({
  formData,
  cutoffOptions,
  onUpdateField,
}: {
  formData: CreateQualityProfileInput
  cutoffOptions: CreateQualityProfileInput['items']
  onUpdateField: (field: keyof CreateQualityProfileInput, value: unknown) => void
}) {
  const selectedLabel =
    cutoffOptions.find((i) => i.quality.id === formData.cutoff)?.quality.name ?? 'Select cutoff'

  return (
    <div className="space-y-2">
      <Label>Cutoff</Label>
      <Select
        value={formData.cutoff.toString()}
        onValueChange={(v) => onUpdateField('cutoff', Number.parseInt(v ?? '0'))}
      >
        <SelectTrigger className="h-8 text-sm">{selectedLabel}</SelectTrigger>
        <SelectContent>
          {cutoffOptions.map((item) => (
            <SelectItem key={item.quality.id} value={item.quality.id.toString()}>
              {item.quality.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function AttributeFiltersSection({
  formData,
  hdrOptions,
  attributeOptions,
  conflictingAttributes,
  onUpdateItemMode,
}: {
  formData: CreateQualityProfileInput
  hdrOptions: string[]
  attributeOptions: AttributeOptions | undefined
  conflictingAttributes: Set<string>
  onUpdateItemMode: (field: AttributeSettingsField, value: string, mode: AttributeMode) => void
}) {
  return (
    <div className="space-y-2">
      <Label>Attribute Filters</Label>

      <CompactAttributeSection
        label="HDR Format"
        settings={formData.hdrSettings}
        options={hdrOptions}
        isConflicting={conflictingAttributes.has('HDR')}
        onItemModeChange={(value, mode) => onUpdateItemMode('hdrSettings', value, mode)}
      />

      <CompactAttributeSection
        label="Video Codec"
        settings={formData.videoCodecSettings}
        options={attributeOptions?.videoCodecs ?? []}
        isConflicting={conflictingAttributes.has('Video Codec')}
        onItemModeChange={(value, mode) => onUpdateItemMode('videoCodecSettings', value, mode)}
      />

      <CompactAttributeSection
        label="Audio Codec"
        settings={formData.audioCodecSettings}
        options={attributeOptions?.audioCodecs ?? []}
        isConflicting={conflictingAttributes.has('Audio Codec')}
        onItemModeChange={(value, mode) => onUpdateItemMode('audioCodecSettings', value, mode)}
      />

      <CompactAttributeSection
        label="Audio Channels"
        settings={formData.audioChannelSettings}
        options={attributeOptions?.audioChannels ?? []}
        isConflicting={conflictingAttributes.has('Audio Channels')}
        onItemModeChange={(value, mode) => onUpdateItemMode('audioChannelSettings', value, mode)}
      />
    </div>
  )
}
