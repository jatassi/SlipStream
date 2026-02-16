import { useCallback, useEffect, useRef } from 'react'

import {
  AlertTriangle,
  Bug,
  ChevronsDown,
  ChevronsUp,
  CircleX,
  Download,
  Info,
  Pause,
  Play,
  Search,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Input } from '@/components/ui/input'
import { useGlobalLoading } from '@/hooks'
import { useDownloadLogFile, useLogs } from '@/hooks/use-logs'
import { cn } from '@/lib/utils'
import { ALL_LOG_LEVELS, useLogsStore } from '@/stores/logs'
import type { LogEntry, LogLevel } from '@/types/logs'

const LEVEL_COLORS: Record<string, string> = {
  debug: 'text-blue-400',
  info: 'text-green-400',
  warn: 'text-yellow-400',
  error: 'text-red-400',
  fatal: 'text-red-600 font-bold',
}

const LEVEL_OPTIONS: { value: LogLevel; label: string; icon: typeof Bug }[] = [
  { value: 'debug', label: 'Debug', icon: Bug },
  { value: 'info', label: 'Info', icon: Info },
  { value: 'warn', label: 'Warning', icon: AlertTriangle },
  { value: 'error', label: 'Error', icon: CircleX },
]

function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3,
    })
  } catch {
    return timestamp
  }
}

function formatFields(fields: Record<string, unknown> | undefined): string {
  if (!fields || Object.keys(fields).length === 0) {
    return ''
  }
  return Object.entries(fields)
    .map(([k, v]) => `${k}=${JSON.stringify(v)}`)
    .join(' ')
}

function LogEntryRow({ entry }: { entry: LogEntry }) {
  const fields = formatFields(entry.fields)
  return (
    <div className="flex gap-2 rounded px-1 hover:bg-zinc-900">
      <span className="shrink-0 text-zinc-500">{formatTimestamp(entry.timestamp)}</span>
      <span className={cn('w-12 shrink-0 uppercase', LEVEL_COLORS[entry.level] || 'text-zinc-400')}>
        {entry.level.slice(0, 5).padEnd(5)}
      </span>
      {entry.component ? (
        <span className="shrink-0 text-cyan-400">[{entry.component}]</span>
      ) : null}
      <span className="text-zinc-100">{entry.message}</span>
      {fields ? <span className="text-zinc-500">{fields}</span> : null}
    </div>
  )
}

type LogsToolbarProps = {
  searchText: string
  onSearchChange: (value: string) => void
  filterLevels: LogLevel[]
  onToggleLevel: (level: LogLevel) => void
  onResetLevels: () => void
  autoScroll: boolean
  onToggleAutoScroll: () => void
  isPaused: boolean
  onTogglePaused: () => void
  onClear: () => void
  onDownload: () => void
  downloadPending: boolean
}

function LogsToolbar({
  searchText, onSearchChange, filterLevels, onToggleLevel, onResetLevels,
  autoScroll, onToggleAutoScroll, isPaused, onTogglePaused, onClear, onDownload, downloadPending,
}: LogsToolbarProps) {
  return (
    <div className="mb-4 flex items-center gap-2">
      <div className="relative max-w-xs flex-1">
        <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
        <Input
          placeholder="Search logs..."
          value={searchText}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-8"
        />
      </div>
      <FilterDropdown
        options={LEVEL_OPTIONS}
        selected={filterLevels}
        onToggle={onToggleLevel}
        onReset={onResetLevels}
        label="Levels"
      />
      <div className="flex-1" />
      <Button variant="outline" size="sm" onClick={onToggleAutoScroll} title={autoScroll ? 'Disable auto-scroll' : 'Enable auto-scroll'}>
        {autoScroll ? <ChevronsDown className="size-4" /> : <ChevronsUp className="size-4" />}
      </Button>
      <Button variant="outline" size="sm" onClick={onTogglePaused} title={isPaused ? 'Resume streaming' : 'Pause streaming'}>
        {isPaused ? <Play className="size-4" /> : <Pause className="size-4" />}
      </Button>
      <Button variant="outline" size="sm" onClick={onClear} title="Clear logs">
        <Trash2 className="size-4" />
      </Button>
      <Button variant="outline" size="sm" onClick={onDownload} disabled={downloadPending} title="Download log file">
        <Download className="size-4" />
      </Button>
    </div>
  )
}

function useLogScroll(entries: LogEntry[]) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const store = useLogsStore()

  useEffect(() => {
    if (store.autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [entries.length, store.autoScroll])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) {return}
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
    if (atBottom && !store.autoScroll) {store.setAutoScroll(true)}
    else if (!atBottom && store.autoScroll) {store.setAutoScroll(false)}
  }, [store])

  return { scrollRef, handleScroll }
}

type LogScrollPanelProps = {
  entries: LogEntry[]
  emptyMessage: string
  scrollRef: React.RefObject<HTMLDivElement | null>
  onScroll: () => void
}

function LogScrollPanel({ entries, emptyMessage, scrollRef, onScroll }: LogScrollPanelProps) {
  return (
    <div ref={scrollRef} onScroll={onScroll} className="flex-1 overflow-auto rounded-md border bg-zinc-950 font-mono text-xs">
      {entries.length === 0 ? (
        <div className="text-muted-foreground flex h-full items-center justify-center">
          {emptyMessage}
        </div>
      ) : (
        <div className="space-y-px p-2">
          {entries.map((entry) => (
            <LogEntryRow key={entry.id} entry={entry} />
          ))}
        </div>
      )}
    </div>
  )
}

function LogsStatusBar({ entryCount, isPaused, autoScroll }: { entryCount: number; isPaused: boolean; autoScroll: boolean }) {
  return (
    <div className="text-muted-foreground mt-2 flex items-center gap-4 text-xs">
      <span>{entryCount} entries</span>
      {isPaused ? <span className="text-yellow-500">Streaming paused</span> : null}
      {!autoScroll && <span>Auto-scroll disabled</span>}
    </div>
  )
}

export function LogsPage() {
  const globalLoading = useGlobalLoading()
  const { isLoading: queryLoading } = useLogs()
  const isLoading = queryLoading || globalLoading
  const downloadMutation = useDownloadLogFile()

  const store = useLogsStore()
  const filteredEntries = store.getFilteredEntries()
  const allSelected = store.filterLevels.length === ALL_LOG_LEVELS.length
  const { scrollRef, handleScroll } = useLogScroll(filteredEntries)
  const emptyMessage = store.searchText || !allSelected ? 'No logs match your filters' : 'No logs yet'

  const handleDownload = async () => {
    try {
      await downloadMutation.mutateAsync()
      toast.success('Log file downloaded')
    } catch {
      toast.error('Failed to download log file')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="System Logs" description="Real-time log streaming" />
        <LoadingState variant="list" count={10} />
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100vh-120px)] flex-col">
      <PageHeader title="System Logs" description="Real-time log streaming" />
      <LogsToolbar
        searchText={store.searchText}
        onSearchChange={store.setSearchText}
        filterLevels={store.filterLevels}
        onToggleLevel={store.toggleFilterLevel}
        onResetLevels={store.resetFilterLevels}
        autoScroll={store.autoScroll}
        onToggleAutoScroll={store.toggleAutoScroll}
        isPaused={store.isPaused}
        onTogglePaused={store.togglePaused}
        onClear={store.clear}
        onDownload={handleDownload}
        downloadPending={downloadMutation.isPending}
      />
      <LogScrollPanel entries={filteredEntries} emptyMessage={emptyMessage} scrollRef={scrollRef} onScroll={handleScroll} />
      <LogsStatusBar entryCount={filteredEntries.length} isPaused={store.isPaused} autoScroll={store.autoScroll} />
    </div>
  )
}
