import { useState } from 'react'

import { Layers } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import type { MasterToggleCardProps } from '@/components/settings/sections/master-toggle-card'
import { MasterToggleCard } from '@/components/settings/sections/master-toggle-card'
import { useVersionSlotsSection } from '@/components/settings/sections/use-version-slots-section'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { SettingsList } from '@/components/settings/settings-list'
import { SlotDebugPanel } from '@/components/slots'
import type { Slot } from '@/types'

import { SlotEditDialog } from './slot-edit-dialog'
import { SlotMigrationModals } from './slot-migration-modals'

type SectionState = ReturnType<typeof useVersionSlotsSection>

export function VersionSlotsPage() {
  const section = useVersionSlotsSection()
  const back = usePushBack()
  const [editingSlot, setEditingSlot] = useState<Slot | null>(null)

  return (
    <Screen title="Version Slots" back={back}>
      <VersionSlotsBody section={section} onEdit={setEditingSlot} />
      <SlotEditDialog
        slot={editingSlot}
        section={section}
        onClose={() => setEditingSlot(null)}
      />
      <SlotMigrationModals section={section} />
    </Screen>
  )
}

function VersionSlotsBody({
  section,
  onEdit,
}: {
  section: SectionState
  onEdit: (slot: Slot) => void
}) {
  if (section.isError) {
    return (
      <div className="px-screen">
        <ErrorState onRetry={section.handleRetry} />
      </div>
    )
  }

  return (
    <>
      <div className="px-screen mb-7">
        <MasterToggleCard {...buildToggleProps(section)} />
      </div>
      <SettingsList
        header="Slots"
        state={{
          isLoading: section.isLoading,
          isError: false,
          isEmpty: !section.slots?.length,
          refetch: section.handleRetry,
        }}
        empty="No version slots"
      >
        {section.slots?.map((slot) => (
          <SlotRow key={slot.id} slot={slot} onEdit={onEdit} />
        ))}
      </SettingsList>
      {section.developerMode ? (
        <div className="px-screen">
          <SlotDebugPanel />
        </div>
      ) : null}
    </>
  )
}

function SlotRow({ slot, onEdit }: { slot: Slot; onEdit: (slot: Slot) => void }) {
  return (
    <SettingsItemRow
      leading={
        <IconTile className={slot.enabled ? 'bg-violet-600' : 'bg-zinc-600'}>
          <Layers />
        </IconTile>
      }
      title={slot.name}
      subtitle={slotSubtitle(slot)}
      detail={slot.enabled ? 'Active' : 'Disabled'}
      onOpen={() => onEdit(slot)}
      openLabel={`Edit ${slot.name}`}
    />
  )
}

function slotSubtitle(slot: Slot): string {
  const profile = slot.qualityProfile?.name ?? 'No quality profile'
  if (slot.fileCount !== undefined && slot.fileCount > 0) {
    return `${profile} · ${slot.fileCount} files`
  }
  return profile
}

function buildToggleProps(s: SectionState): MasterToggleCardProps {
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
