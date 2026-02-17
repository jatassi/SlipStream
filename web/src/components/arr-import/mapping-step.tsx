import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { useQualityProfiles } from '@/hooks/use-quality-profiles'
import { useRootFolders } from '@/hooks/use-root-folders'
import type {
  ImportMappings,
  SourceQualityProfile,
  SourceRootFolder,
} from '@/types/arr-import'

import { useMappingState } from './use-mapping-state'

type MappingStepProps = {
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  onMappingsComplete: (mappings: ImportMappings) => void
}

export function MappingStep({
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
  mappingState,
  sourceRootFolders,
  sourceQualityProfiles,
  targetRootFolders,
  targetQualityProfiles,
  onMappingsComplete,
}: {
  mappingState: ReturnType<typeof useMappingState>
  sourceRootFolders: SourceRootFolder[]
  sourceQualityProfiles: SourceQualityProfile[]
  targetRootFolders: { id: number; path: string }[]
  targetQualityProfiles: { id: number; name: string }[]
  onMappingsComplete: (mappings: ImportMappings) => void
}) {
  const canProceed = mappingState.allRootFoldersMapped && mappingState.allProfilesMapped

  return (
    <div className="space-y-6">
      <RootFolderMappingSection
        sourceRootFolders={sourceRootFolders}
        targetRootFolders={targetRootFolders}
        rootFolderMapping={mappingState.rootFolderMapping}
        setRootFolderMapping={mappingState.setRootFolderMapping}
      />

      <QualityProfileMappingSection
        sourceQualityProfiles={sourceQualityProfiles}
        targetQualityProfiles={targetQualityProfiles}
        qualityProfileMapping={mappingState.qualityProfileMapping}
        setQualityProfileMapping={mappingState.setQualityProfileMapping}
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
  sourceRootFolders,
  targetRootFolders,
  rootFolderMapping,
  setRootFolderMapping,
}: {
  sourceRootFolders: SourceRootFolder[]
  targetRootFolders: { id: number; path: string }[]
  rootFolderMapping: Record<string, number>
  setRootFolderMapping: React.Dispatch<React.SetStateAction<Record<string, number>>>
}) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-medium">Root Folder Mapping</h3>
        <p className="text-muted-foreground text-sm">
          Map each source root folder to a SlipStream root folder
        </p>
      </div>

      <div className="space-y-3">
        {sourceRootFolders.map((sourceFolder) => (
          <RootFolderMappingRow
            key={sourceFolder.id}
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
  sourceFolder,
  targetRootFolders,
  selectedTargetId,
  onSelect,
}: {
  sourceFolder: SourceRootFolder
  targetRootFolders: { id: number; path: string }[]
  selectedTargetId: number | undefined
  onSelect: (targetId: number) => void
}) {
  const selectedPath = targetRootFolders.find((f) => f.id === selectedTargetId)?.path ?? 'Select folder...'

  return (
    <div className="grid grid-cols-2 items-center gap-4">
      <div className="space-y-1">
        <Label className="text-xs font-normal">Source</Label>
        <div className="border-input bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm">
          {sourceFolder.path}
        </div>
      </div>

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
          <SelectTrigger>{selectedPath}</SelectTrigger>
          <SelectContent>
            {targetRootFolders.map((folder) => (
              <SelectItem key={folder.id} value={folder.id.toString()}>
                {folder.path}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

function QualityProfileMappingSection({
  sourceQualityProfiles,
  targetQualityProfiles,
  qualityProfileMapping,
  setQualityProfileMapping,
}: {
  sourceQualityProfiles: SourceQualityProfile[]
  targetQualityProfiles: { id: number; name: string }[]
  qualityProfileMapping: Record<number, number>
  setQualityProfileMapping: React.Dispatch<React.SetStateAction<Record<number, number>>>
}) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-medium">Quality Profile Mapping</h3>
        <p className="text-muted-foreground text-sm">
          Map each source quality profile to a SlipStream quality profile
        </p>
      </div>

      <div className="space-y-3">
        {sourceQualityProfiles.map((sourceProfile) => (
          <QualityProfileMappingRow
            key={sourceProfile.id}
            sourceProfile={sourceProfile}
            targetQualityProfiles={targetQualityProfiles}
            selectedTargetId={qualityProfileMapping[sourceProfile.id]}
            onSelect={(targetId) =>
              setQualityProfileMapping((prev) => ({ ...prev, [sourceProfile.id]: targetId }))
            }
          />
        ))}
      </div>
    </div>
  )
}

function QualityProfileMappingRow({
  sourceProfile,
  targetQualityProfiles,
  selectedTargetId,
  onSelect,
}: {
  sourceProfile: SourceQualityProfile
  targetQualityProfiles: { id: number; name: string }[]
  selectedTargetId: number | undefined
  onSelect: (targetId: number) => void
}) {
  const selectedName = targetQualityProfiles.find((p) => p.id === selectedTargetId)?.name ?? 'Select profile...'

  return (
    <div className="grid grid-cols-2 items-center gap-4">
      <div className="space-y-1">
        <Label className="text-xs font-normal">Source</Label>
        <div className="border-input bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm">
          {sourceProfile.name}
        </div>
      </div>

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
          <SelectTrigger>{selectedName}</SelectTrigger>
          <SelectContent>
            {targetQualityProfiles.map((profile) => (
              <SelectItem key={profile.id} value={profile.id.toString()}>
                {profile.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
