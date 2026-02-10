import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import {
  Film, Tv, History, Trash2, Search, AlertCircle,
  FileEdit, RefreshCw, ArrowUpCircle, PackageCheck,
  ChevronDown, ChevronRight, ArrowRight,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { useHistory, useClearHistory } from '@/hooks'
import { formatRelativeTime } from '@/lib/formatters'
import { eventTypeColors, eventTypeLabels, filterableEventTypes, isUpgradeEvent } from '@/lib/history-utils'
import { toast } from 'sonner'
import type { HistoryEventType, HistoryEntry } from '@/types'

type MediaFilter = 'all' | 'movie' | 'episode'

const DATE_PRESETS = [
  { value: 'all', label: 'All Time' },
  { value: 'today', label: 'Today' },
  { value: '7days', label: 'Last 7 Days' },
  { value: '30days', label: 'Last 30 Days' },
  { value: '90days', label: 'Last 90 Days' },
] as const

type DatePreset = typeof DATE_PRESETS[number]['value']

function getAfterDate(preset: DatePreset): string | undefined {
  if (preset === 'all') return undefined
  const now = new Date()
  switch (preset) {
    case 'today':
      now.setHours(0, 0, 0, 0)
      return now.toISOString()
    case '7days':
      now.setDate(now.getDate() - 7)
      now.setHours(0, 0, 0, 0)
      return now.toISOString()
    case '30days':
      now.setDate(now.getDate() - 30)
      now.setHours(0, 0, 0, 0)
      return now.toISOString()
    case '90days':
      now.setDate(now.getDate() - 90)
      now.setHours(0, 0, 0, 0)
      return now.toISOString()
  }
}

function EventIcon({ eventType }: { eventType: HistoryEventType }) {
  switch (eventType) {
    case 'autosearch_download':
      return <Search className="size-3 mr-1" />
    case 'autosearch_failed':
    case 'import_failed':
      return <AlertCircle className="size-3 mr-1" />
    case 'imported':
      return <PackageCheck className="size-3 mr-1" />
    case 'file_renamed':
      return <FileEdit className="size-3 mr-1" />
    case 'status_changed':
      return <RefreshCw className="size-3 mr-1" />
    default:
      return null
  }
}

function QualityChange({ from, to }: { from: string; to: string }) {
  return (
    <Badge variant="secondary" className="gap-1 text-[10px] px-1.5 py-0">
      <span className="text-yellow-500">{from}</span>
      <ArrowRight className="size-2.5 text-muted-foreground" />
      <span className="text-green-500">{to}</span>
    </Badge>
  )
}

function getDetailsText(item: HistoryEntry): string {
  const data = item.data as Record<string, unknown> | undefined
  if (!data) return item.source || '-'

  switch (item.eventType) {
    case 'autosearch_download': {
      const release = (data.releaseName as string) || item.source || '-'
      if (data.isUpgrade && data.newQuality) return `${release} (upgrade to ${data.newQuality})`
      if (data.isUpgrade) return `${release} (upgrade)`
      return release
    }
    case 'autosearch_failed':
      return (data.error as string) || 'Search failed'
    case 'imported':
      return (data.finalFilename as string) || (data.originalFilename as string) || item.source || '-'
    case 'import_failed':
      return (data.error as string) || 'Import failed'
    case 'status_changed': {
      const from = data.from as string
      const to = data.to as string
      if (from && to) return `${from} → ${to}`
      return item.source || '-'
    }
    case 'file_renamed': {
      const oldName = data.old_filename as string
      const newName = data.new_filename as string
      if (oldName && newName) return `${oldName} → ${newName}`
      return item.source || '-'
    }
    default:
      return item.source || '-'
  }
}

function DetailsContent({ item }: { item: HistoryEntry }) {
  const data = item.data as Record<string, unknown> | undefined

  if (item.eventType === 'imported' && data?.isUpgrade && data.previousQuality && data.newQuality) {
    return (
      <QualityChange from={data.previousQuality as string} to={data.newQuality as string} />
    )
  }

  return <>{getDetailsText(item)}</>
}

function getDetailRows(item: HistoryEntry): { label: string; value: string }[] {
  const data = item.data as Record<string, unknown> | undefined
  if (!data) return []

  const rows: { label: string; value: string }[] = []

  switch (item.eventType) {
    case 'autosearch_download':
      if (data.releaseName) rows.push({ label: 'Release', value: data.releaseName as string })
      if (data.indexer) rows.push({ label: 'Indexer', value: data.indexer as string })
      if (data.clientName) rows.push({ label: 'Client', value: data.clientName as string })
      if (data.downloadId) rows.push({ label: 'Download ID', value: data.downloadId as string })
      if (data.source) rows.push({ label: 'Trigger', value: data.source as string })
      if (data.oldQuality) rows.push({ label: 'Previous Quality', value: data.oldQuality as string })
      if (data.newQuality) rows.push({ label: 'New Quality', value: data.newQuality as string })
      break
    case 'autosearch_failed':
      if (data.error) rows.push({ label: 'Error', value: data.error as string })
      if (data.indexer) rows.push({ label: 'Indexer', value: data.indexer as string })
      break
    case 'imported':
      if (data.sourcePath) rows.push({ label: 'Source', value: data.sourcePath as string })
      if (data.destinationPath) rows.push({ label: 'Destination', value: data.destinationPath as string })
      if (data.originalFilename) rows.push({ label: 'Original', value: data.originalFilename as string })
      if (data.finalFilename) rows.push({ label: 'Final', value: data.finalFilename as string })
      if (data.clientName) rows.push({ label: 'Client', value: data.clientName as string })
      if (data.codec) rows.push({ label: 'Codec', value: data.codec as string })
      if (data.size) rows.push({ label: 'Size', value: formatFileSize(data.size as number) })
      if (data.previousFile) rows.push({ label: 'Previous File', value: data.previousFile as string })
      if (data.error) rows.push({ label: 'Error', value: data.error as string })
      break
    case 'import_failed':
      if (data.error) rows.push({ label: 'Error', value: data.error as string })
      if (data.sourcePath) rows.push({ label: 'Source', value: data.sourcePath as string })
      break
    case 'status_changed':
      if (data.from) rows.push({ label: 'From', value: data.from as string })
      if (data.to) rows.push({ label: 'To', value: data.to as string })
      if (data.reason) rows.push({ label: 'Reason', value: data.reason as string })
      break
    case 'file_renamed':
      if (data.source_path) rows.push({ label: 'Old Path', value: data.source_path as string })
      if (data.destination_path) rows.push({ label: 'New Path', value: data.destination_path as string })
      break
  }

  return rows
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function getPaginationPages(current: number, total: number): (number | 'ellipsis')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)

  const pages: (number | 'ellipsis')[] = [1]

  if (current > 3) pages.push('ellipsis')

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) pages.push(i)

  if (current < total - 2) pages.push('ellipsis')

  pages.push(total)
  return pages
}

function ExpandedRow({ item }: { item: HistoryEntry }) {
  const rows = getDetailRows(item)
  if (rows.length === 0) return null

  return (
    <TableRow className="bg-muted/30 hover:bg-muted/30">
      <TableCell colSpan={6} className="py-3 px-8">
        <div className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
          {rows.map((row, i) => (
            <div key={i} className="contents">
              <span className="text-muted-foreground font-medium">{row.label}</span>
              <span className="truncate" title={row.value}>{row.value}</span>
            </div>
          ))}
        </div>
      </TableCell>
    </TableRow>
  )
}

function MediaTitle({ item }: { item: HistoryEntry }) {
  const isMovie = item.mediaType === 'movie'
  const title = item.mediaTitle || `${item.mediaType} #${item.mediaId}`
  const qualifier = isMovie
    ? (item.year ? String(item.year) : undefined)
    : item.mediaQualifier

  const content = (
    <>
      <span className="font-medium">{title}</span>
      {qualifier && <span className="text-muted-foreground ml-1.5">{qualifier}</span>}
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

export function HistoryPage() {
  const [eventTypes, setEventTypes] = useState<HistoryEventType[]>(
    filterableEventTypes.map((et) => et.value),
  )
  const [mediaType, setMediaType] = useState<MediaFilter>('all')
  const [datePreset, setDatePreset] = useState<DatePreset>('all')
  const [page, setPage] = useState(1)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const mediaTypeParam = mediaType === 'all' ? undefined : mediaType
  const allEventTypesSelected = eventTypes.length >= filterableEventTypes.length

  const { data: history, isLoading, isError, refetch } = useHistory({
    eventType: allEventTypesSelected ? undefined : eventTypes.join(','),
    mediaType: mediaTypeParam,
    after: getAfterDate(datePreset),
    page,
    pageSize: 50,
  })

  const clearMutation = useClearHistory()

  const handleClearHistory = async () => {
    try {
      await clearMutation.mutateAsync()
      toast.success('History cleared')
    } catch {
      toast.error('Failed to clear history')
    }
  }

  const handleToggleEventType = (value: HistoryEventType) => {
    setEventTypes((prev) => {
      const has = prev.includes(value)
      return has ? prev.filter((v) => v !== value) : [...prev, value]
    })
    setPage(1)
  }

  const handleMediaTypeChange = (v: string) => {
    setMediaType(v as MediaFilter)
    setPage(1)
  }

  const handleDatePresetChange = (v: string | null) => {
    if (v) {
      setDatePreset(v as DatePreset)
      setPage(1)
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="History" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="History" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="History"
        description="View past activity and events"
        actions={
          <ConfirmDialog
            trigger={
              <Button variant="destructive">
                <Trash2 className="size-4 mr-2" />
                Clear History
              </Button>
            }
            title="Clear history"
            description="Are you sure you want to clear all history? This action cannot be undone."
            confirmLabel="Clear"
            variant="destructive"
            onConfirm={handleClearHistory}
          />
        }
      />

      {/* Media type tabs + filters */}
      <div className="flex items-center justify-between mb-4">
        <Tabs value={mediaType} onValueChange={handleMediaTypeChange}>
          <TabsList>
            <TabsTrigger
              value="all"
              className="px-4 data-active:bg-white data-active:text-black data-active:glow-media-sm"
            >
              All
            </TabsTrigger>
            <TabsTrigger
              value="movie"
              className="data-active:bg-white data-active:text-black data-active:glow-movie"
            >
              <Film className="size-4 mr-1.5" />
              Movies
            </TabsTrigger>
            <TabsTrigger
              value="episode"
              className="data-active:bg-white data-active:text-black data-active:glow-tv"
            >
              <Tv className="size-4 mr-1.5" />
              Series
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex items-center gap-3">
          <FilterDropdown
            options={filterableEventTypes}
            selected={eventTypes}
            onToggle={handleToggleEventType}
            onReset={() => setEventTypes(filterableEventTypes.map((e) => e.value))}
            label="Events"
          />

          <Select
            value={datePreset}
            onValueChange={handleDatePresetChange}
          >
            <SelectTrigger className="w-auto">
              {DATE_PRESETS.find(p => p.value === datePreset)?.label}
            </SelectTrigger>
            <SelectContent>
              {DATE_PRESETS.map((preset) => (
                <SelectItem key={preset.value} value={preset.value}>
                  {preset.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* History table */}
      <Card>
        <CardContent className="p-0">
          {!history?.items?.length ? (
            <EmptyState
              icon={<History className="size-8" />}
              title="No history"
              description="Activity history will appear here"
              className="py-8"
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8"></TableHead>
                  <TableHead className="w-12"></TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Details</TableHead>
                  <TableHead>Date</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history.items.map((item) => {
                  const isExpanded = expandedId === item.id
                  const hasDetails = getDetailRows(item).length > 0
                  const isMovie = item.mediaType === 'movie'

                  return (
                    <>
                      <TableRow
                        key={item.id}
                        className={cn(
                          'cursor-pointer',
                          isMovie
                            ? 'hover:bg-movie-500/5'
                            : 'hover:bg-tv-500/5',
                          isExpanded && 'bg-muted/20',
                        )}
                        onClick={() => hasDetails && setExpandedId(isExpanded ? null : item.id)}
                      >
                        <TableCell className="w-8 pr-0">
                          {hasDetails && (
                            isExpanded
                              ? <ChevronDown className="size-3.5 text-muted-foreground" />
                              : <ChevronRight className="size-3.5 text-muted-foreground" />
                          )}
                        </TableCell>
                        <TableCell className="w-12">
                          {isMovie ? (
                            <Film className="size-4 text-movie-500" />
                          ) : (
                            <Tv className="size-4 text-tv-500" />
                          )}
                        </TableCell>
                        <TableCell>
                          <MediaTitle item={item} />
                          {item.quality && (
                            <Badge variant="outline" className="ml-2 text-[10px] px-1.5 py-0">
                              {item.quality}
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            <Badge variant={eventTypeColors[item.eventType] || 'outline'} className="flex items-center w-fit">
                              <EventIcon eventType={item.eventType} />
                              {eventTypeLabels[item.eventType] || item.eventType}
                            </Badge>
                            {isUpgradeEvent(item.data as Record<string, unknown>) && (
                              <Badge variant="outline" className="flex items-center gap-0.5 text-[10px] px-1.5 py-0">
                                <ArrowUpCircle className="size-2.5" />
                                Upgrade
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="max-w-xs truncate text-sm text-muted-foreground" title={getDetailsText(item)}>
                          <DetailsContent item={item} />
                        </TableCell>
                        <TableCell className="text-muted-foreground whitespace-nowrap">
                          {formatRelativeTime(item.createdAt)}
                        </TableCell>
                      </TableRow>
                      {isExpanded && <ExpandedRow key={`${item.id}-detail`} item={item} />}
                    </>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {history && history.totalPages > 1 && (
        <Pagination className="mt-4">
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className={page === 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
              />
            </PaginationItem>
            {getPaginationPages(page, history.totalPages).map((p, i) =>
              p === 'ellipsis' ? (
                <PaginationItem key={`ellipsis-${i}`}>
                  <PaginationEllipsis />
                </PaginationItem>
              ) : (
                <PaginationItem key={p}>
                  <PaginationLink
                    isActive={p === page}
                    onClick={() => setPage(p)}
                    className="cursor-pointer"
                  >
                    {p}
                  </PaginationLink>
                </PaginationItem>
              )
            )}
            <PaginationItem>
              <PaginationNext
                onClick={() => setPage((p) => Math.min(history.totalPages, p + 1))}
                className={page === history.totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  )
}
