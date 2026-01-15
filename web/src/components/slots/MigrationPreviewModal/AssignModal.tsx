import { useState, useEffect } from 'react'
import { Layers } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import type { AssignModalProps } from './types'

export function AssignModal({ open, onOpenChange, slots, selectedCount, onAssign }: AssignModalProps) {
  const [selectedSlotId, setSelectedSlotId] = useState<string>('')

  useEffect(() => {
    if (!open) {
      setSelectedSlotId('')
    }
  }, [open])

  const handleAssign = () => {
    if (!selectedSlotId) return
    const slot = slots.find(s => s.id === parseInt(selectedSlotId))
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
            Select a slot to assign {selectedCount} selected file{selectedCount !== 1 ? 's' : ''} to
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Label htmlFor="slot-select" className="text-sm font-medium mb-2 block">
            Select Slot
          </Label>
          <Select value={selectedSlotId} onValueChange={(value) => setSelectedSlotId(value ?? '')}>
            <SelectTrigger id="slot-select" className="w-full">
              {selectedSlotId ? slots.find(s => s.id === parseInt(selectedSlotId))?.name : 'Choose a slot...'}
            </SelectTrigger>
            <SelectContent className="min-w-[var(--trigger-width)]">
              {slots.map(slot => (
                <SelectItem key={slot.id} value={slot.id.toString()}>
                  <div className="flex items-center gap-2">
                    <Layers className="size-4" />
                    <span>{slot.name}</span>
                    {slot.qualityProfile && (
                      <span className="text-muted-foreground text-xs">
                        ({slot.qualityProfile.name})
                      </span>
                    )}
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleAssign} disabled={!selectedSlotId}>
            <Layers className="size-4 mr-2" />
            Assign
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
