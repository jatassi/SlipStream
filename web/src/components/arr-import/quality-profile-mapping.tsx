import { ArrowRight } from 'lucide-react'

import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { SourceQualityProfile } from '@/types/arr-import'

type QualityProfileMappingSectionProps = {
  label: string
  sourceQualityProfiles: SourceQualityProfile[]
  targetQualityProfiles: { id: number; name: string }[]
  qualityProfileMapping: Record<number, number>
  setQualityProfileMapping: React.Dispatch<React.SetStateAction<Record<number, number>>>
  profileEnabled: Record<number, boolean>
  setProfileEnabled: React.Dispatch<React.SetStateAction<Record<number, boolean>>>
}

export function QualityProfileMappingSection({
  label,
  sourceQualityProfiles,
  targetQualityProfiles,
  qualityProfileMapping,
  setQualityProfileMapping,
  profileEnabled,
  setProfileEnabled,
}: QualityProfileMappingSectionProps) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-medium">Quality Profile Mapping</h3>
        <p className="text-muted-foreground text-sm">
          Map each {label} quality profile to a SlipStream quality profile
        </p>
      </div>

      <div className="space-y-3">
        {sourceQualityProfiles.map((sourceProfile) => (
          <QualityProfileMappingRow
            key={sourceProfile.id}
            label={label}
            sourceProfile={sourceProfile}
            targetQualityProfiles={targetQualityProfiles}
            selectedTargetId={qualityProfileMapping[sourceProfile.id]}
            enabled={profileEnabled[sourceProfile.id] ?? false}
            onToggle={(checked) =>
              setProfileEnabled((prev) => ({ ...prev, [sourceProfile.id]: checked }))
            }
            onSelect={(targetId) =>
              setQualityProfileMapping((prev) => ({ ...prev, [sourceProfile.id]: targetId }))
            }
          />
        ))}
      </div>
    </div>
  )
}

function ProfileTargetSelect({
  enabled,
  selectedTargetId,
  targetQualityProfiles,
  onToggle,
  onSelect,
}: {
  enabled: boolean
  selectedTargetId: number | undefined
  targetQualityProfiles: { id: number; name: string }[]
  onToggle: (checked: boolean) => void
  onSelect: (targetId: number) => void
}) {
  const selectedName = enabled
    ? (targetQualityProfiles.find((p) => p.id === selectedTargetId)?.name ?? 'Select profile...')
    : 'Skipped'

  return (
    <div className="flex items-center gap-2">
      <Checkbox checked={enabled} onCheckedChange={onToggle} />
      <Select
        value={enabled ? (selectedTargetId?.toString() ?? '') : ''}
        onValueChange={(value) => {
          if (value) {
            onSelect(Number.parseInt(value, 10))
          }
        }}
        disabled={!enabled}
      >
        <SelectTrigger className={enabled ? '' : 'opacity-50'}>{selectedName}</SelectTrigger>
        <SelectContent>
          {targetQualityProfiles.map((profile) => (
            <SelectItem key={profile.id} value={profile.id.toString()}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function QualityProfileMappingRow({
  label,
  sourceProfile,
  targetQualityProfiles,
  selectedTargetId,
  enabled,
  onToggle,
  onSelect,
}: {
  label: string
  sourceProfile: SourceQualityProfile
  targetQualityProfiles: { id: number; name: string }[]
  selectedTargetId: number | undefined
  enabled: boolean
  onToggle: (checked: boolean) => void
  onSelect: (targetId: number) => void
}) {
  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4">
      <div className="space-y-1">
        <Label className="text-xs font-normal">{label} Quality Profile</Label>
        <div className="border-input bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm">
          {sourceProfile.name}
        </div>
      </div>

      <ArrowRight className="text-muted-foreground mt-5 size-4" />

      <div className="space-y-1">
        <Label className="text-xs font-normal">SlipStream</Label>
        <ProfileTargetSelect
          enabled={enabled}
          selectedTargetId={selectedTargetId}
          targetQualityProfiles={targetQualityProfiles}
          onToggle={onToggle}
          onSelect={onSelect}
        />
      </div>
    </div>
  )
}
