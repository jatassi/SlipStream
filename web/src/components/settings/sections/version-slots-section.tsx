import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import {
  DryRunModal,
  ResolveConfigModal,
  ResolveNamingModal,
  SlotDebugPanel,
} from '@/components/slots'
import type { RootFolder, Slot, SlotNamingValidation } from '@/types'

import type { MasterToggleCardProps } from './master-toggle-card'
import { MasterToggleCard } from './master-toggle-card'
import { SlotCard } from './slot-card'
import type { ValidationResult } from './use-version-slots-section'
import { useVersionSlotsSection } from './use-version-slots-section'

function getUsedProfileIds(slots: Slot[], excludeId: number): number[] {
  const ids: number[] = []
  for (const s of slots) {
    if (s.id !== excludeId && s.qualityProfileId !== null) {
      ids.push(s.qualityProfileId)
    }
  }
  return ids
}

type SlotGridProps = {
  slots: Slot[] | undefined
  profiles: { id: number; name: string }[]
  movieRootFolders: RootFolder[]
  tvRootFolders: RootFolder[]
  isSlotUpdating: boolean
  onEnabledChange: (slot: Slot, enabled: boolean) => void
  onNameChange: (slot: Slot, name: string) => void
  onProfileChange: (slot: Slot, profileId: string) => void
  onRootFolderChange: (slot: Slot, mediaType: 'movie' | 'tv', rootFolderId: string) => void
}

function SlotGrid(props: SlotGridProps) {
  const { slots } = props
  if (!slots) {
    return <div className="grid gap-4 md:grid-cols-3" />
  }

  return (
    <div className="grid gap-4 md:grid-cols-3">
      {slots.map((slot) => (
        <SlotCard
          key={slot.id}
          slot={slot}
          profiles={props.profiles}
          usedProfileIds={getUsedProfileIds(slots, slot.id)}
          movieRootFolders={props.movieRootFolders}
          tvRootFolders={props.tvRootFolders}
          onEnabledChange={(enabled) => props.onEnabledChange(slot, enabled)}
          onNameChange={(name) => props.onNameChange(slot, name)}
          onProfileChange={(profileId) => props.onProfileChange(slot, profileId)}
          onRootFolderChange={(mediaType, rootFolderId) =>
            props.onRootFolderChange(slot, mediaType, rootFolderId)
          }
          isUpdating={props.isSlotUpdating}
          showToggle={slot.slotNumber === 3}
        />
      ))}
    </div>
  )
}

type SectionModalsProps = {
  resolveConfigOpen: boolean
  resolveNamingOpen: boolean
  dryRunOpen: boolean
  validationResult: ValidationResult
  namingValidation: SlotNamingValidation | null
  onResolveConfigChange: (open: boolean) => void
  onResolveNamingChange: (open: boolean) => void
  onDryRunChange: (open: boolean) => void
  onValidate: () => void
  onValidateNaming: () => void
  onMigrationComplete: () => void
  onMigrationFailed: (error: string) => void
}

function SectionModals(props: SectionModalsProps) {
  return (
    <>
      <ResolveConfigModal
        open={props.resolveConfigOpen}
        onOpenChange={props.onResolveConfigChange}
        conflicts={props.validationResult?.conflicts ?? []}
        onResolved={props.onValidate}
      />
      <ResolveNamingModal
        open={props.resolveNamingOpen}
        onOpenChange={props.onResolveNamingChange}
        missingMovieTokens={props.namingValidation?.movieValidation.missingTokens}
        missingEpisodeTokens={props.namingValidation?.episodeValidation.missingTokens}
        onResolved={props.onValidateNaming}
      />
      <DryRunModal
        open={props.dryRunOpen}
        onOpenChange={props.onDryRunChange}
        onMigrationComplete={props.onMigrationComplete}
        onMigrationFailed={props.onMigrationFailed}
      />
    </>
  )
}

function buildToggleProps(s: ReturnType<typeof useVersionSlotsSection>): MasterToggleCardProps {
  return {
    settingsEnabled: s.settings?.enabled ?? false,
    multiVersionEnabled: s.multiVersionEnabled,
    enabledSlotCount: s.enabledSlotCount,
    isTogglePending: s.isTogglePending,
    configurationReady: s.configurationReady,
    migrationError: s.migrationError,
    infoCardDismissed: s.infoCardDismissed,
    validationResult: s.validationResult,
    namingValidation: s.namingValidation,
    isValidatePending: s.isValidatePending,
    isValidateNamingPending: s.isValidateNamingPending,
    onToggleMultiVersion: (enabled: boolean) => void s.handleToggleMultiVersion(enabled),
    onDismissInfo: () => s.setInfoCardDismissed(true),
    onDismissMigrationError: () => s.setMigrationError(null),
    onBeginDryRun: () => s.setDryRunOpen(true),
    onValidate: () => void s.handleValidate(),
    onValidateNaming: () => void s.handleValidateNaming(),
    onResolveConfig: () => s.setResolveConfigOpen(true),
    onResolveNaming: () => s.setResolveNamingOpen(true),
  }
}

export function VersionSlotsSection() {
  const s = useVersionSlotsSection()

  if (s.isLoading) {
    return <LoadingState variant="list" count={3} />
  }
  if (s.isError) {
    return <ErrorState onRetry={s.handleRetry} />
  }

  return (
    <div className="space-y-6">
      <MasterToggleCard {...buildToggleProps(s)} />
      <SlotGrid
        slots={s.slots}
        profiles={s.profiles}
        movieRootFolders={s.movieRootFolders}
        tvRootFolders={s.tvRootFolders}
        isSlotUpdating={s.isSlotUpdating}
        onEnabledChange={s.handleSlotEnabledChange}
        onNameChange={s.handleSlotNameChange}
        onProfileChange={s.handleSlotProfileChange}
        onRootFolderChange={s.handleSlotRootFolderChange}
      />
      {s.developerMode ? <SlotDebugPanel /> : null}
      <SectionModals
        resolveConfigOpen={s.resolveConfigOpen}
        resolveNamingOpen={s.resolveNamingOpen}
        dryRunOpen={s.dryRunOpen}
        validationResult={s.validationResult}
        namingValidation={s.namingValidation}
        onResolveConfigChange={s.setResolveConfigOpen}
        onResolveNamingChange={s.setResolveNamingOpen}
        onDryRunChange={s.setDryRunOpen}
        onValidate={() => void s.handleValidate()}
        onValidateNaming={() => void s.handleValidateNaming()}
        onMigrationComplete={s.handleMigrationComplete}
        onMigrationFailed={(error) => s.setMigrationError(error)}
      />
    </div>
  )
}
