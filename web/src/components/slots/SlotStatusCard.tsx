import { Eye, EyeOff, ArrowUpCircle, ArrowDownCircle, AlertCircle, CheckCircle, XCircle, Clock, Search, Zap } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { MediaStatus, SlotStatus } from '@/types'
import { cn } from '@/lib/utils'

interface SlotStatusCardProps {
  status: MediaStatus | undefined
  isLoading: boolean
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  onManualSearch?: (slotId: number) => void
  onAutoSearch?: (slotId: number) => void
  isUpdating?: boolean
  isSearching?: number | null // slotId currently being searched
}

export function SlotStatusCard({
  status,
  isLoading,
  onToggleMonitored,
  onManualSearch,
  onAutoSearch,
  isUpdating,
  isSearching,
}: SlotStatusCardProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Version Slots</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-32 w-full" />
        </CardContent>
      </Card>
    )
  }

  if (!status || !status.slotStatuses || status.slotStatuses.length === 0) {
    return null
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            Version Slots
            <SlotSummaryBadges status={status} />
          </CardTitle>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Slot</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Quality</TableHead>
              <TableHead>Actions</TableHead>
              <TableHead className="text-right">Monitored</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {status.slotStatuses.map((slot) => (
              <SlotStatusRow
                key={slot.slotId}
                slot={slot}
                onToggleMonitored={onToggleMonitored}
                onManualSearch={onManualSearch}
                onAutoSearch={onAutoSearch}
                isUpdating={isUpdating}
                isSearching={isSearching === slot.slotId}
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

interface SlotSummaryBadgesProps {
  status: MediaStatus
}

function SlotSummaryBadges({ status }: SlotSummaryBadgesProps) {
  const hasMissing = status.slotStatuses.some(s => s.status === 'missing')
  const hasUpgradable = status.slotStatuses.some(s => s.status === 'upgradable')
  const hasFailed = status.slotStatuses.some(s => s.status === 'failed')
  const hasDownloading = status.slotStatuses.some(s => s.status === 'downloading')
  const allGood = status.slotStatuses.every(s => s.status === 'available' || !s.monitored)

  return (
    <div className="flex items-center gap-1.5">
      {hasFailed && (
        <Tooltip>
          <TooltipTrigger>
            <Badge variant="destructive" className="gap-1">
              <XCircle className="size-3" />
              Failed
            </Badge>
          </TooltipTrigger>
          <TooltipContent>One or more slots have failed downloads</TooltipContent>
        </Tooltip>
      )}
      {hasMissing && (
        <Tooltip>
          <TooltipTrigger>
            <Badge variant="destructive" className="gap-1">
              <AlertCircle className="size-3" />
              Missing
            </Badge>
          </TooltipTrigger>
          <TooltipContent>One or more monitored slots are empty</TooltipContent>
        </Tooltip>
      )}
      {hasDownloading && (
        <Tooltip>
          <TooltipTrigger>
            <Badge className="gap-1 bg-blue-600 hover:bg-blue-600 text-white">
              <ArrowDownCircle className="size-3" />
              Downloading
            </Badge>
          </TooltipTrigger>
          <TooltipContent>One or more slots are downloading</TooltipContent>
        </Tooltip>
      )}
      {hasUpgradable && !hasMissing && (
        <Tooltip>
          <TooltipTrigger>
            <Badge variant="secondary" className="gap-1">
              <ArrowUpCircle className="size-3" />
              Upgrade Available
            </Badge>
          </TooltipTrigger>
          <TooltipContent>
            One or more files are below the quality cutoff
          </TooltipContent>
        </Tooltip>
      )}
      {allGood && status.filledSlots > 0 && (
        <Tooltip>
          <TooltipTrigger>
            <Badge variant="outline" className="gap-1 border-green-500 text-green-500">
              <CheckCircle className="size-3" />
              Complete
            </Badge>
          </TooltipTrigger>
          <TooltipContent>All monitored slots are filled at target quality</TooltipContent>
        </Tooltip>
      )}
    </div>
  )
}

interface SlotStatusRowProps {
  slot: SlotStatus
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  onManualSearch?: (slotId: number) => void
  onAutoSearch?: (slotId: number) => void
  isUpdating?: boolean
  isSearching?: boolean
}

function SlotStatusRow({
  slot,
  onToggleMonitored,
  onManualSearch,
  onAutoSearch,
  isUpdating,
  isSearching,
}: SlotStatusRowProps) {
  const handleToggle = (checked: boolean) => {
    onToggleMonitored?.(slot.slotId, checked)
  }

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          <span className="font-medium">{slot.slotName}</span>
          <span className="text-muted-foreground text-xs">#{slot.slotNumber}</span>
        </div>
      </TableCell>
      <TableCell>
        <SlotStatusBadge slot={slot} />
      </TableCell>
      <TableCell>
        {slot.currentQuality ? (
          <Badge variant="outline">{slot.currentQuality}</Badge>
        ) : (
          <span className="text-muted-foreground text-sm">-</span>
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onManualSearch?.(slot.slotId)}
                  disabled={isSearching}
                />
              }
            >
              <Search className="size-4" />
            </TooltipTrigger>
            <TooltipContent>Manual search for {slot.slotName}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onAutoSearch?.(slot.slotId)}
                  disabled={isSearching}
                />
              }
            >
              <Zap className={cn('size-4', isSearching && 'animate-pulse')} />
            </TooltipTrigger>
            <TooltipContent>Auto search for {slot.slotName}</TooltipContent>
          </Tooltip>
        </div>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          <Label htmlFor={`slot-${slot.slotId}-monitored`} className="sr-only">
            Monitor slot {slot.slotName}
          </Label>
          <Switch
            id={`slot-${slot.slotId}-monitored`}
            checked={slot.monitored}
            onCheckedChange={handleToggle}
            disabled={isUpdating}
          />
          {slot.monitored ? (
            <Eye className="size-4 text-muted-foreground" />
          ) : (
            <EyeOff className="size-4 text-muted-foreground" />
          )}
        </div>
      </TableCell>
    </TableRow>
  )
}

interface SlotStatusBadgeProps {
  slot: SlotStatus
}

function SlotStatusBadge({ slot }: SlotStatusBadgeProps) {
  if (!slot.monitored && slot.status !== 'available' && slot.status !== 'upgradable') {
    return (
      <Badge variant="outline" className="gap-1 text-muted-foreground">
        <EyeOff className="size-3" />
        Not Monitored
      </Badge>
    )
  }

  switch (slot.status) {
    case 'failed':
      return (
        <Badge variant="destructive" className="gap-1 bg-red-900/50 border border-red-500 text-red-400">
          <XCircle className="size-3" />
          Failed
        </Badge>
      )
    case 'downloading':
      return (
        <Badge className="gap-1 bg-blue-600 hover:bg-blue-600 text-white">
          <ArrowDownCircle className="size-3" />
          Downloading
        </Badge>
      )
    case 'missing':
      return (
        <Badge variant="destructive" className="gap-1">
          <AlertCircle className="size-3" />
          Missing
        </Badge>
      )
    case 'upgradable':
      return (
        <Badge variant="secondary" className="gap-1">
          <ArrowUpCircle className="size-3" />
          Upgrade
        </Badge>
      )
    case 'available':
      return (
        <Badge variant="outline" className={cn('gap-1', 'border-green-500 text-green-500')}>
          <CheckCircle className="size-3" />
          OK
        </Badge>
      )
    case 'unreleased':
      return (
        <Badge variant="outline" className="gap-1 border-amber-500 text-amber-500">
          <Clock className="size-3" />
          Unreleased
        </Badge>
      )
    default:
      return (
        <Badge variant="outline" className="text-muted-foreground">
          Empty
        </Badge>
      )
  }
}
