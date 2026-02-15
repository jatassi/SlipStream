import { Link } from '@tanstack/react-router'
import {
  AlertCircle,
  ArrowRight,
  ArrowUpCircle,
  ChevronDown,
  ChevronRight,
  FileEdit,
  Film,
  PackageCheck,
  RefreshCw,
  Search,
  Tv,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatRelativeTime } from '@/lib/formatters'
import {
  eventTypeColors,
  eventTypeLabels,
  isUpgradeEvent,
} from '@/lib/history-utils'
import { cn } from '@/lib/utils'
import type { HistoryEntry, HistoryEventType } from '@/types'

import { getDetailRows, getDetailsText } from './history-utils'

function EventIcon({ eventType }: { eventType: HistoryEventType }) {
  const iconMap: Partial<Record<HistoryEventType, React.ReactNode>> = {
    autosearch_download: <Search className="mr-1 size-3" />,
    autosearch_failed: <AlertCircle className="mr-1 size-3" />,
    import_failed: <AlertCircle className="mr-1 size-3" />,
    imported: <PackageCheck className="mr-1 size-3" />,
    file_renamed: <FileEdit className="mr-1 size-3" />,
    status_changed: <RefreshCw className="mr-1 size-3" />,
  }
  return iconMap[eventType] ?? null
}

function QualityChange({ from, to }: { from: string; to: string }) {
  return (
    <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
      <span className="text-yellow-500">{from}</span>
      <ArrowRight className="text-muted-foreground size-2.5" />
      <span className="text-green-500">{to}</span>
    </Badge>
  )
}

function DetailsContent({ item }: { item: HistoryEntry }) {
  const data = item.data as Record<string, unknown> | undefined

  if (item.eventType === 'imported' && data?.isUpgrade && data.previousQuality && data.newQuality) {
    return <QualityChange from={data.previousQuality as string} to={data.newQuality as string} />
  }

  return <>{getDetailsText(item)}</>
}

function MediaTitle({ item }: { item: HistoryEntry }) {
  const isMovie = item.mediaType === 'movie'
  const title = item.mediaTitle ?? `${item.mediaType} #${item.mediaId}`
  const movieYear = item.year ? String(item.year) : undefined
  const qualifier = isMovie ? movieYear : item.mediaQualifier

  const content = (
    <>
      <span className="font-medium">{title}</span>
      {qualifier ? <span className="text-muted-foreground ml-1.5">{qualifier}</span> : null}
    </>
  )

  if (isMovie) {
    return (
      <Link
        to="/movies/$id"
        params={{ id: String(item.mediaId) }}
        className="hover:text-movie-500 transition-colors hover:underline"
      >
        {content}
      </Link>
    )
  }

  if (item.seriesId) {
    return (
      <Link
        to="/series/$id"
        params={{ id: String(item.seriesId) }}
        className="hover:text-tv-500 transition-colors hover:underline"
      >
        {content}
      </Link>
    )
  }

  return <span>{content}</span>
}

export function ExpandedRow({ item }: { item: HistoryEntry }) {
  const rows = getDetailRows(item)
  if (rows.length === 0) {
    return null
  }

  return (
    <TableRow className="bg-muted/30 hover:bg-muted/30">
      <TableCell colSpan={6} className="px-8 py-3">
        <div className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
          {rows.map((row) => (
            <div key={`${row.label}-${row.value}`} className="contents">
              <span className="text-muted-foreground font-medium">{row.label}</span>
              <span className="truncate" title={row.value}>
                {row.value}
              </span>
            </div>
          ))}
        </div>
      </TableCell>
    </TableRow>
  )
}

function ExpandChevron({ hasDetails, isExpanded }: { hasDetails: boolean; isExpanded: boolean }) {
  if (!hasDetails) {
    return null
  }
  if (isExpanded) {
    return <ChevronDown className="text-muted-foreground size-3.5" />
  }
  return <ChevronRight className="text-muted-foreground size-3.5" />
}

function MediaIcon({ isMovie }: { isMovie: boolean }) {
  if (isMovie) {
    return <Film className="text-movie-500 size-4" />
  }
  return <Tv className="text-tv-500 size-4" />
}

function EventBadge({ item }: { item: HistoryEntry }) {
  return (
    <div className="flex items-center gap-1.5">
      <Badge
        variant={eventTypeColors[item.eventType]}
        className="flex w-fit items-center"
      >
        <EventIcon eventType={item.eventType} />
        {eventTypeLabels[item.eventType]}
      </Badge>
      {isUpgradeEvent(item.data as Record<string, unknown>) && (
        <Badge
          variant="outline"
          className="flex items-center gap-0.5 px-1.5 py-0 text-[10px]"
        >
          <ArrowUpCircle className="size-2.5" />
          Upgrade
        </Badge>
      )}
    </div>
  )
}

type HistoryRowProps = {
  item: HistoryEntry
  isExpanded: boolean
  onToggle: (id: number, hasDetails: boolean) => void
}

export function HistoryRow({ item, isExpanded, onToggle }: HistoryRowProps) {
  const hasDetails = getDetailRows(item).length > 0
  const isMovie = item.mediaType === 'movie'

  return (
    <TableRow
      className={cn(
        'cursor-pointer',
        isMovie ? 'hover:bg-movie-500/5' : 'hover:bg-tv-500/5',
        isExpanded && 'bg-muted/20',
      )}
      onClick={() => onToggle(item.id, hasDetails)}
    >
      <TableCell className="w-8 pr-0">
        <ExpandChevron hasDetails={hasDetails} isExpanded={isExpanded} />
      </TableCell>
      <TableCell className="w-12">
        <MediaIcon isMovie={isMovie} />
      </TableCell>
      <TableCell>
        <MediaTitle item={item} />
        {item.quality ? (
          <Badge variant="outline" className="ml-2 px-1.5 py-0 text-[10px]">
            {item.quality}
          </Badge>
        ) : null}
      </TableCell>
      <TableCell>
        <EventBadge item={item} />
      </TableCell>
      <TableCell
        className="text-muted-foreground max-w-xs truncate text-sm"
        title={getDetailsText(item)}
      >
        <DetailsContent item={item} />
      </TableCell>
      <TableCell className="text-muted-foreground whitespace-nowrap">
        {formatRelativeTime(item.createdAt)}
      </TableCell>
    </TableRow>
  )
}

const TABLE_HEADERS = (
  <TableHeader>
    <TableRow>
      <TableHead className="w-8" />
      <TableHead className="w-12" />
      <TableHead>Title</TableHead>
      <TableHead>Event</TableHead>
      <TableHead>Details</TableHead>
      <TableHead>Date</TableHead>
    </TableRow>
  </TableHeader>
)

export function HistoryTableSkeleton() {
  return (
    <Table>
      {TABLE_HEADERS}
      <TableBody>
        {Array.from({ length: 12 }, (_, i) => (
          <TableRow key={i}>
            <TableCell className="w-8 pr-0">
              <Skeleton className="size-3.5" />
            </TableCell>
            <TableCell className="w-12">
              <Skeleton className="size-4 rounded-full" />
            </TableCell>
            <TableCell>
              <div className="flex items-center gap-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-10" />
              </div>
            </TableCell>
            <TableCell>
              <Skeleton className="h-5 w-20 rounded-full" />
            </TableCell>
            <TableCell>
              <Skeleton className="h-4 w-48" />
            </TableCell>
            <TableCell>
              <Skeleton className="h-4 w-16" />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

type HistoryTableProps = {
  items: HistoryEntry[]
  expandedId: number | null
  onToggleExpanded: (id: number, hasDetails: boolean) => void
}

export function HistoryTable({ items, expandedId, onToggleExpanded }: HistoryTableProps) {
  return (
    <Table>
      {TABLE_HEADERS}
      <TableBody>
        {items.map((item) => (
          <HistoryRowGroup
            key={item.id}
            item={item}
            isExpanded={expandedId === item.id}
            onToggle={onToggleExpanded}
          />
        ))}
      </TableBody>
    </Table>
  )
}

function HistoryRowGroup({
  item,
  isExpanded,
  onToggle,
}: HistoryRowProps) {
  return (
    <>
      <HistoryRow item={item} isExpanded={isExpanded} onToggle={onToggle} />
      {isExpanded ? <ExpandedRow item={item} /> : null}
    </>
  )
}
