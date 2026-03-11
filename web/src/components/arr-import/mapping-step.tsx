import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useQualityProfiles } from '@/hooks/use-quality-profiles'
import { useRootFolders } from '@/hooks/use-root-folders'
import type {
  ImportMappings,
  SourceQualityProfile,
  SourceRootFolder,
  SourceType,
} from '@/types/arr-import'
import type { RootFolder } from '@/types/root-folder'

import { QualityProfileMappingSection } from './quality-profile-mapping'
import { RootFolderMappingSection } from './root-folder-mapping'
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
  const { data: targetQualityProfiles, isLoading: isLoadingProfiles } = useQualityProfiles(sourceType === 'radarr' ? 'movie' : 'tv')

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
