import { Eye, EyeOff, ArrowUpCircle, AlertCircle, CheckCircle, Search, Zap, Loader2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { SlotStatus } from '@/types'
import { cn } from '@/lib/utils'

interface EpisodeSlotRowProps {
  slotStatuses: SlotStatus[]
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  onManualSearch?: (slotId: number) => void
  onAutoSearch?: (slotId: number) => void
  isUpdating?: boolean
  isSearching?: number | null
}

export function EpisodeSlotRow({
  slotStatuses,
  onToggleMonitored,
  onManualSearch,
  onAutoSearch,
  isUpdating,
  isSearching,
}: EpisodeSlotRowProps) {
  if (!slotStatuses || slotStatuses.length === 0) {
    return null
  }

  return (
    <div className="space-y-1 py-2 px-3 bg-muted/30 rounded-md">
      {slotStatuses.map((slot) => (
        <CompactSlotItem
          key={slot.slotId}
          slot={slot}
          onToggleMonitored={onToggleMonitored}
          onManualSearch={onManualSearch}
          onAutoSearch={onAutoSearch}
          isUpdating={isUpdating}
          isSearching={isSearching === slot.slotId}
        />
      ))}
    </div>
  )
}

interface CompactSlotItemProps {
  slot: SlotStatus
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  onManualSearch?: (slotId: number) => void
  onAutoSearch?: (slotId: number) => void
  isUpdating?: boolean
  isSearching?: boolean
}

function CompactSlotItem({
  slot,
  onToggleMonitored,
  onManualSearch,
  onAutoSearch,
  isUpdating,
  isSearching,
}: CompactSlotItemProps) {
  const handleToggle = (checked: boolean) => {
    onToggleMonitored?.(slot.slotId, checked)
  }

  return (
    <div className="flex items-center justify-between gap-2 py-1 text-xs">
      <div className="flex items-center gap-2 min-w-0">
        <span className="font-medium shrink-0">{slot.slotName}</span>
        <CompactSlotBadge slot={slot} />
        {slot.currentQuality && (
          <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4">
            {slot.currentQuality}
          </Badge>
        )}
      </div>

      <div className="flex items-center gap-1 shrink-0">
        {isSearching ? (
          <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
        ) : (
          <>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-6"
                    onClick={() => onManualSearch?.(slot.slotId)}
                    disabled={isUpdating}
                  />
                }
              >
                <Search className="size-3" />
              </TooltipTrigger>
              <TooltipContent>Manual search</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-6"
                    onClick={() => onAutoSearch?.(slot.slotId)}
                    disabled={isUpdating}
                  />
                }
              >
                <Zap className="size-3" />
              </TooltipTrigger>
              <TooltipContent>Auto search</TooltipContent>
            </Tooltip>
          </>
        )}
        <Switch
          checked={slot.monitored}
          onCheckedChange={handleToggle}
          disabled={isUpdating}
          className="scale-75"
        />
        {slot.monitored ? (
          <Eye className="size-3 text-muted-foreground" />
        ) : (
          <EyeOff className="size-3 text-muted-foreground" />
        )}
      </div>
    </div>
  )
}

function CompactSlotBadge({ slot }: { slot: SlotStatus }) {
  if (slot.isMissing) {
    return (
      <Badge variant="destructive" className="text-[10px] px-1.5 py-0 h-4 gap-0.5">
        <AlertCircle className="size-2.5" />
        Missing
      </Badge>
    )
  }

  if (slot.needsUpgrade) {
    return (
      <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4 gap-0.5">
        <ArrowUpCircle className="size-2.5" />
        Upgrade
      </Badge>
    )
  }

  if (slot.hasFile) {
    return (
      <Badge
        variant="outline"
        className={cn('text-[10px] px-1.5 py-0 h-4 gap-0.5', 'border-green-500 text-green-500')}
      >
        <CheckCircle className="size-2.5" />
        OK
      </Badge>
    )
  }

  if (!slot.monitored) {
    return (
      <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 gap-0.5 text-muted-foreground">
        <EyeOff className="size-2.5" />
        Not Monitored
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 text-muted-foreground">
      Empty
    </Badge>
  )
}
