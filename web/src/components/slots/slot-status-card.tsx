import {
  AlertCircle,
  ArrowDownCircle,
  ArrowUpCircle,
  CheckCircle,
  Clock,
  EyeOff,
  XCircle,
} from 'lucide-react'

import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import type { MediaStatus, SlotStatus } from '@/types'

type SlotStatusCardProps = {
  status: MediaStatus | undefined
  isLoading: boolean
  movieId: number
  movieTitle: string
  qualityProfileId: number
  tmdbId?: number
  imdbId?: string
  year?: number
  slotQualityProfiles?: Record<number, number>
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  isUpdating?: boolean
}

export function SlotStatusCard(props: SlotStatusCardProps) {
  const { status, isLoading } = props

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

  if (!status?.slotStatuses || status.slotStatuses.length === 0) {
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
        <SlotStatusTable {...props} slotStatuses={status.slotStatuses} />
      </CardContent>
    </Card>
  )
}

type SlotStatusTableProps = Pick<
  SlotStatusCardProps,
  'movieId' | 'movieTitle' | 'qualityProfileId' | 'tmdbId' | 'imdbId' | 'year' | 'slotQualityProfiles' | 'onToggleMonitored' | 'isUpdating'
> & {
  slotStatuses: SlotStatus[]
}

function SlotStatusTable(props: SlotStatusTableProps) {
  const { slotStatuses, movieId, movieTitle, qualityProfileId, slotQualityProfiles, onToggleMonitored, isUpdating } = props

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Slot</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Quality</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {slotStatuses.map((slot) => (
          <SlotStatusRow
            key={slot.slotId}
            slot={slot}
            movieId={movieId}
            movieTitle={movieTitle}
            qualityProfileId={slotQualityProfiles?.[slot.slotId] ?? qualityProfileId}
            tmdbId={props.tmdbId}
            imdbId={props.imdbId}
            year={props.year}
            onToggleMonitored={onToggleMonitored}
            isUpdating={isUpdating}
          />
        ))}
      </TableBody>
    </Table>
  )
}

type SummaryBadgeConfig = {
  variant: 'destructive' | 'secondary' | 'outline'
  className?: string
  icon: React.ComponentType<{ className?: string }>
  label: string
  tooltip: string
}

const SUMMARY_BADGES: (SummaryBadgeConfig & { key: string })[] = [
  { key: 'failed', variant: 'destructive', icon: XCircle, label: 'Failed', tooltip: 'One or more slots have failed downloads' },
  { key: 'missing', variant: 'destructive', icon: AlertCircle, label: 'Missing', tooltip: 'One or more monitored slots are empty' },
  { key: 'downloading', variant: 'secondary', className: 'bg-blue-600 text-white hover:bg-blue-600', icon: ArrowDownCircle, label: 'Downloading', tooltip: 'One or more slots are downloading' },
  { key: 'upgradable', variant: 'secondary', icon: ArrowUpCircle, label: 'Upgrade Available', tooltip: 'One or more files are below the quality cutoff' },
  { key: 'complete', variant: 'outline', className: 'border-green-500 text-green-500', icon: CheckCircle, label: 'Complete', tooltip: 'All monitored slots are filled at target quality' },
]

function getVisibleBadgeKeys(status: MediaStatus): Set<string> {
  const hasStatus = (s: string) => status.slotStatuses.some((slot) => slot.status === s)
  const hasMissing = hasStatus('missing')
  const allGood = status.slotStatuses.every((s) => s.status === 'available' || !s.monitored)

  const keys = new Set<string>()
  if (hasStatus('failed')) {
    keys.add('failed')
  }
  if (hasMissing) {
    keys.add('missing')
  }
  if (hasStatus('downloading')) {
    keys.add('downloading')
  }
  if (hasStatus('upgradable') && !hasMissing) {
    keys.add('upgradable')
  }
  if (allGood && status.filledSlots > 0) {
    keys.add('complete')
  }
  return keys
}

function SlotSummaryBadges({ status }: { status: MediaStatus }) {
  const visibleKeys = getVisibleBadgeKeys(status)

  return (
    <div className="flex items-center gap-1.5">
      {SUMMARY_BADGES.filter((b) => visibleKeys.has(b.key)).map((b) => (
        <SummaryBadge key={b.key} config={b} />
      ))}
    </div>
  )
}

function SummaryBadge({ config }: { config: SummaryBadgeConfig }) {
  const Icon = config.icon
  return (
    <Tooltip>
      <TooltipTrigger>
        <Badge variant={config.variant} className={cn('gap-1', config.className)}>
          <Icon className="size-3" />
          {config.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent>{config.tooltip}</TooltipContent>
    </Tooltip>
  )
}

type SlotStatusRowProps = {
  slot: SlotStatus
  movieId: number
  movieTitle: string
  qualityProfileId: number
  tmdbId?: number
  imdbId?: string
  year?: number
  onToggleMonitored?: (slotId: number, monitored: boolean) => void
  isUpdating?: boolean
}

function SlotStatusRow(props: SlotStatusRowProps) {
  const { slot, onToggleMonitored, isUpdating } = props

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
      <TableCell className="text-right">
        <SlotActions
          slot={slot}
          movieId={props.movieId}
          movieTitle={props.movieTitle}
          qualityProfileId={props.qualityProfileId}
          tmdbId={props.tmdbId}
          imdbId={props.imdbId}
          year={props.year}
          onToggleMonitored={onToggleMonitored}
          isUpdating={isUpdating}
        />
      </TableCell>
    </TableRow>
  )
}

function SlotActions(props: SlotStatusRowProps) {
  const { slot, movieId, movieTitle, onToggleMonitored, isUpdating } = props

  return (
    <div className="flex items-center justify-end">
      <MediaSearchMonitorControls
        mediaType="movie-slot"
        movieId={movieId}
        slotId={slot.slotId}
        title={`${movieTitle} — ${slot.slotName}`}
        theme="movie"
        size="sm"
        monitored={slot.monitored}
        onMonitoredChange={(m) => onToggleMonitored?.(slot.slotId, m)}
        monitorDisabled={isUpdating}
        qualityProfileId={props.qualityProfileId}
        tmdbId={props.tmdbId}
        imdbId={props.imdbId}
        year={props.year}
      />
    </div>
  )
}

type StatusBadgeConfig = {
  variant: 'destructive' | 'secondary' | 'outline'
  className?: string
  icon: React.ComponentType<{ className?: string }>
  label: string
}

const STATUS_BADGE_MAP: Record<SlotStatus['status'], StatusBadgeConfig> = {
  failed: {
    variant: 'destructive',
    className: 'border border-red-500 bg-red-900/50 text-red-400',
    icon: XCircle,
    label: 'Failed',
  },
  downloading: {
    variant: 'secondary',
    className: 'bg-blue-600 text-white hover:bg-blue-600',
    icon: ArrowDownCircle,
    label: 'Downloading',
  },
  missing: {
    variant: 'destructive',
    icon: AlertCircle,
    label: 'Missing',
  },
  upgradable: {
    variant: 'secondary',
    icon: ArrowUpCircle,
    label: 'Upgrade',
  },
  available: {
    variant: 'outline',
    className: 'border-green-500 text-green-500',
    icon: CheckCircle,
    label: 'OK',
  },
  unreleased: {
    variant: 'outline',
    className: 'border-amber-500 text-amber-500',
    icon: Clock,
    label: 'Unreleased',
  },
}

function SlotStatusBadge({ slot }: { slot: SlotStatus }) {
  if (!slot.monitored && slot.status !== 'available' && slot.status !== 'upgradable') {
    return (
      <Badge variant="outline" className="text-muted-foreground gap-1">
        <EyeOff className="size-3" />
        Not Monitored
      </Badge>
    )
  }

  const config = STATUS_BADGE_MAP[slot.status]
  const Icon = config.icon
  return (
    <Badge variant={config.variant} className={cn('gap-1', config.className)}>
      <Icon className="size-3" />
      {config.label}
    </Badge>
  )
}
