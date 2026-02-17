import { ArrowRight, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { useQualityProfiles } from '@/hooks/use-quality-profiles'
import { useRootFolders } from '@/hooks/use-root-folders'
import type {
  ImportMappings,
  SourceQualityProfile,
  SourceRootFolder,
  SourceType,
} from '@/types/arr-import'
import type { RootFolder } from '@/types/root-folder'

import { useMappingState } from './use-mapping-state'

type MappingStepProps = {
  sourceType: SourceType
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  onMappingsComplete: (mappings: ImportMappings) => void
}

function sourceLabel(sourceType: SourceType): string {
  return sourceType === 'radarr' ? 'Radarr' : 'Sonarr'
}

export function MappingStep({
  sourceType,
  sourceRootFolders,
  sourceQualityProfiles,
  onMappingsComplete,
}: MappingStepProps) {
  const { data: targetRootFolders, isLoading: isLoadingRootFolders } = useRootFolders()
  const { data: targetQualityProfiles, isLoading: isLoadingProfiles } = useQualityProfiles()

  const mappingState = useMappingState({
    sourceRootFolders,
    sourceQualityProfiles,
    targetRootFolders,
    targetQualityProfiles,
  })

  if (isLoadingRootFolders || isLoadingProfiles) {
    return <MappingLoadingState />
  }

  if (!targetRootFolders || !targetQualityProfiles) {
    return <MappingErrorState />
  }

  return (
    <MappingForm
      sourceType={sourceType}
      mappingState={mappingState}
      sourceRootFolders={sourceRootFolders}
      sourceQualityProfiles={sourceQualityProfiles}
      targetRootFolders={targetRootFolders}
      targetQualityProfiles={targetQualityProfiles}
      onMappingsComplete={onMappingsComplete}
    />
  )
}

function MappingLoadingState() {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <Loader2 className="text-muted-foreground mb-4 size-8 animate-spin" />
      <p className="text-muted-foreground text-sm">Loading SlipStream configuration...</p>
    </div>
  )
}

function MappingErrorState() {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <p className="text-destructive text-sm">Failed to load SlipStream configuration</p>
    </div>
  )
}

function MappingForm({
  sourceType,
  mappingState,
  sourceRootFolders,
  sourceQualityProfiles,
  targetRootFolders,
  targetQualityProfiles,
  onMappingsComplete,
}: {
  sourceType: SourceType
  mappingState: ReturnType<typeof useMappingState>
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  targetRootFolders: RootFolder[]
  targetQualityProfiles: { id: number; name: string }[]
  onMappingsComplete: (mappings: ImportMappings) => void
}) {
  const canProceed = mappingState.allRootFoldersMapped && mappingState.allProfilesMapped
  const label = sourceLabel(sourceType)

  return (
    <div className="space-y-6">
      <RootFolderMappingSection
        label={label}
        sourceRootFolders={sourceRootFolders}
        targetRootFolders={targetRootFolders}
        rootFolderMapping={mappingState.rootFolderMapping}
        setRootFolderMapping={mappingState.setRootFolderMapping}
      />

      <QualityProfileMappingSection
        label={label}
        sourceQualityProfiles={sourceQualityProfiles}
        targetQualityProfiles={targetQualityProfiles}
        qualityProfileMapping={mappingState.qualityProfileMapping}
        setQualityProfileMapping={mappingState.setQualityProfileMapping}
        profileEnabled={mappingState.profileEnabled}
        setProfileEnabled={mappingState.setProfileEnabled}
      />

      <div className="flex justify-end pt-4">
        <Button onClick={() => mappingState.handleNext(onMappingsComplete)} disabled={!canProceed}>
          Next
        </Button>
      </div>
    </div>
  )
}

function RootFolderMappingSection({
  label,
  sourceRootFolders,
  targetRootFolders,
  rootFolderMapping,
  setRootFolderMapping,
}: {
  label: string
  sourceRootFolders: SourceRootFolder[]
  targetRootFolders: RootFolder[]
  rootFolderMapping: Record<string, number>
  setRootFolderMapping: React.Dispatch<React.SetStateAction<Record<string, number>>>
}) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-medium">Root Folder Mapping</h3>
        <p className="text-muted-foreground text-sm">
          Map each {label} root folder to a SlipStream root folder
        </p>
      </div>

      <div className="space-y-3">
        {sourceRootFolders.map((sourceFolder) => (
          <RootFolderMappingRow
            key={sourceFolder.id}
            label={label}
            sourceFolder={sourceFolder}
            targetRootFolders={targetRootFolders}
            selectedTargetId={rootFolderMapping[sourceFolder.path]}
            onSelect={(targetId) =>
              setRootFolderMapping((prev) => ({ ...prev, [sourceFolder.path]: targetId }))
            }
          />
        ))}
      </div>
    </div>
  )
}

function RootFolderMappingRow({
  label,
  sourceFolder,
  targetRootFolders,
  selectedTargetId,
  onSelect,
}: {
  label: string
  sourceFolder: SourceRootFolder
  targetRootFolders: RootFolder[]
  selectedTargetId: number | undefined
  onSelect: (targetId: number) => void
}) {
  const selectedFolder = targetRootFolders.find((f) => f.id === selectedTargetId)
  const triggerDisplay = selectedFolder
    ? `${selectedFolder.name} — ${selectedFolder.path}`
    : 'Select folder...'

  return (
    <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4">
      <div className="space-y-1">
        <Label className="text-xs font-normal">{label} Root Folder</Label>
        <div className="border-input bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm">
          {sourceFolder.path}
        </div>
      </div>

      <ArrowRight className="text-muted-foreground mt-5 size-4" />

      <div className="space-y-1">
        <Label className="text-xs font-normal">SlipStream</Label>
        <Select
          value={selectedTargetId?.toString() ?? ''}
          onValueChange={(value) => {
            if (value) {
              onSelect(Number.parseInt(value, 10))
            }
          }}
        >
          <SelectTrigger>{triggerDisplay}</SelectTrigger>
          <SelectContent>
            {targetRootFolders.map((folder) => (
              <SelectItem key={folder.id} value={folder.id.toString()}>
                {folder.name} — {folder.path}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

function QualityProfileMappingSection({
  label,
  sourceQualityProfiles,
  targetQualityProfiles,
  qualityProfileMapping,
  setQualityProfileMapping,
  profileEnabled,
  setProfileEnabled,
}: {
  label: string
  sourceQualityProfiles: SourceQualityProfile[]
  targetQualityProfiles: { id: number; name: string }[]
  qualityProfileMapping: Record<number, number>
  setQualityProfileMapping: React.Dispatch<React.SetStateAction<Record<number, number>>>
  profileEnabled: Record<number, boolean>
  setProfileEnabled: React.Dispatch<React.SetStateAction<Record<number, boolean>>>
}) {
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
