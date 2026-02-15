import { useState } from 'react'

import { Layers } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { Slot } from '@/types'

import type { AssignModalProps } from './types'

function SlotOption({ slot }: { slot: Slot }) {
  return (
    <div className="flex items-center gap-2">
      <Layers className="size-4" />
      <span>{slot.name}</span>
      {slot.qualityProfile ? (
        <span className="text-muted-foreground text-xs">({slot.qualityProfile.name})</span>
      ) : null}
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
    <div className="py-4">
      <Label htmlFor="slot-select" className="mb-2 block text-sm font-medium">
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

export function AssignModal({
  open,
  onOpenChange,
  slots,
  selectedCount,
  onAssign,
}: AssignModalProps) {
  const [selectedSlotId, setSelectedSlotId] = useState<string>('')
  const [prevOpen, setPrevOpen] = useState(open)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (!open) {
      setSelectedSlotId('')
    }
  }

  const handleAssign = () => {
    const slot = findSlot(slots, selectedSlotId)
    if (slot) {
      onAssign(slot.id, slot.name)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Assign to Slot</DialogTitle>
          <DialogDescription>
            Select a slot to assign {selectedCount} selected file{selectedCount === 1 ? '' : 's'}{' '}
            to
          </DialogDescription>
        </DialogHeader>
        <SlotSelectField
          slots={slots}
          selectedSlotId={selectedSlotId}
          onValueChange={setSelectedSlotId}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleAssign} disabled={!selectedSlotId}>
            <Layers className="mr-2 size-4" />
            Assign
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
