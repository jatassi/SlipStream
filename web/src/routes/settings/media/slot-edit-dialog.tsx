import { SheetPresenter } from '@/components/presenter'
import { SlotCard } from '@/components/settings/sections/slot-card'
import type { useVersionSlotsSection } from '@/components/settings/sections/use-version-slots-section'
import type { Slot } from '@/types'

type SectionState = ReturnType<typeof useVersionSlotsSection>

function usedProfileIds(slots: Slot[], excludeId: number): number[] {
  return slots
    .filter((slot) => slot.id !== excludeId)
    .flatMap((slot) => (slot.qualityProfileId === null ? [] : [slot.qualityProfileId]))
}

export function SlotEditDialog({
  slot,
  section,
  onClose,
}: {
  slot: Slot | null
  section: SectionState
  onClose: () => void
}) {
  if (slot === null) {
    return null
  }

  return (
    <SheetPresenter
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
      title={slot.name}
      description="Configure this version slot."
    >
      <div className="py-2">
        <SlotCard
          slot={slot}
          profiles={section.profiles}
          usedProfileIds={usedProfileIds(section.slots ?? [], slot.id)}
          rootFoldersByModule={section.rootFoldersByModule}
          onEnabledChange={(enabled) => section.handleSlotEnabledChange(slot, enabled)}
          onNameChange={(name) => section.handleSlotNameChange(slot, name)}
          onProfileChange={(profileId) => section.handleSlotProfileChange(slot, profileId)}
          onRootFolderChange={(moduleType, rootFolderId) =>
            section.handleSlotRootFolderChange(slot, moduleType, rootFolderId)
          }
          isUpdating={section.isSlotUpdating}
          showToggle={slot.slotNumber === 3}
        />
      </div>
    </SheetPresenter>
  )
}
