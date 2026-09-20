import { useState } from 'react'

import { Layers } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { Slot } from '@/types'

import type { AssignModalProps } from './types'

function SlotOption({ slot }: { slot: Slot }) {
  return (
    <div className="flex items-center gap-2">
      <Layers className="size-4" />
      <span>{slot.name}</span>
      {slot.qualityProfile !== undefined && <span className="text-muted-foreground text-xs">({slot.qualityProfile.name})</span>}
    </div>
  )
}

function findSlot(slots: Slot[], id: string) {
  return id ? slots.find((s) => s.id === Number.parseInt(id)) : undefined
}

function SlotSelectField({
  slots,
  selectedSlotId,
  onValueChange,
}: {
  slots: Slot[]
  selectedSlotId: string
  onValueChange: (value: string) => void
}) {
  return (
    <div className="py-2">
      <Label htmlFor="slot-select" className="text-body mb-2 block font-medium">
        Select Slot
      </Label>
      <Select value={selectedSlotId} onValueChange={(value) => onValueChange(value ?? '')}>
        <SelectTrigger id="slot-select" className="w-full">
          {findSlot(slots, selectedSlotId)?.name ?? 'Choose a slot...'}
        </SelectTrigger>
        <SelectContent className="min-w-[var(--trigger-width)]">
          {slots.map((slot) => (
            <SelectItem key={slot.id} value={slot.id.toString()}>
              <SlotOption slot={slot} />
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function useAssignModal(open: boolean, slots: Slot[], onAssign: AssignModalProps['onAssign']) {
  const [selectedSlotId, setSelectedSlotId] = useState<string>('')
  const [prevOpen, setPrevOpen] = useState(open)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (!open) {setSelectedSlotId('')}
  }

  const handleAssign = () => {
    const slot = findSlot(slots, selectedSlotId)
    if (slot) {onAssign(slot.id, slot.name)}
  }

  return { selectedSlotId, setSelectedSlotId, handleAssign }
}

export function AssignModal({ open, onOpenChange, slots, selectedCount, onAssign }: AssignModalProps) {
  const s = useAssignModal(open, slots, onAssign)

  return (
    <SheetPresenter
      nested
      open={open}
      onOpenChange={onOpenChange}
      title="Assign to Slot"
      description={`Select a slot to assign ${selectedCount} selected file${selectedCount === 1 ? '' : 's'} to`}
      footer={
        <FormActions
          onCancel={() => onOpenChange(false)}
          confirmLabel="Assign"
          onConfirm={s.handleAssign}
          confirmDisabled={!s.selectedSlotId}
        />
      }
    >
      <SlotSelectField slots={slots} selectedSlotId={s.selectedSlotId} onValueChange={s.setSelectedSlotId} />
    </SheetPresenter>
  )
}
