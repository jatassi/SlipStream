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

import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useDownloadLogFile, useLogs } from '@/hooks/use-logs'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/stores'
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
    <div className="flex gap-2 rounded px-1">
      <span className="shrink-0 text-zinc-500">{formatTimestamp(entry.timestamp)}</span>
      <span className={cn('w-12 shrink-0 uppercase', LEVEL_COLORS[entry.level] || 'text-zinc-400')}>
        {entry.level.slice(0, 5).padEnd(5)}
      </span>
      {Boolean(entry.component) && <span className="shrink-0 text-cyan-400">[{entry.component}]</span>}
      <span className="text-zinc-100">{entry.message}</span>
      {Boolean(fields) && <span className="text-zinc-500">{fields}</span>}
    </div>
  )
}

function ToolbarButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
      className="press flex size-tap shrink-0 items-center justify-center rounded-md text-muted-foreground focus-visible:ring-ring outline-none focus-visible:ring-[3px] disabled:opacity-50"
    >
      {children}
    </button>
  )
}

function LogsSearchField({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="relative min-w-40 flex-1">
      <Search className="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
      <Input
        placeholder="Search logs..."
        aria-label="Search logs"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="pl-8"
      />
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
  searchText,
  onSearchChange,
  filterLevels,
  onToggleLevel,
  onResetLevels,
  autoScroll,
  onToggleAutoScroll,
  isPaused,
  onTogglePaused,
  onClear,
  onDownload,
  downloadPending,
}: LogsToolbarProps) {
  return (
    <div className="px-screen mb-3 flex flex-wrap items-center gap-2">
      <LogsSearchField value={searchText} onChange={onSearchChange} />
      <FilterDropdown
        options={LEVEL_OPTIONS}
        selected={filterLevels}
        onToggle={onToggleLevel}
        onReset={onResetLevels}
        label="Levels"
      />
      <ToolbarButton
        label={autoScroll ? 'Disable auto-scroll' : 'Enable auto-scroll'}
        onClick={onToggleAutoScroll}
      >
        {autoScroll ? <ChevronsDown className="size-4" /> : <ChevronsUp className="size-4" />}
      </ToolbarButton>
      <ToolbarButton label={isPaused ? 'Resume streaming' : 'Pause streaming'} onClick={onTogglePaused}>
        {isPaused ? <Play className="size-4" /> : <Pause className="size-4" />}
      </ToolbarButton>
      <ToolbarButton label="Clear logs" onClick={onClear}>
        <Trash2 className="size-4" />
      </ToolbarButton>
      <ToolbarButton label="Download log file" disabled={downloadPending} onClick={onDownload}>
        <Download className="size-4" />
      </ToolbarButton>
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
    if (!el) {
      return
    }
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
    if (atBottom && !store.autoScroll) {
      store.setAutoScroll(true)
    } else if (!atBottom && store.autoScroll) {
      store.setAutoScroll(false)
    }
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
    <div
      ref={scrollRef}
      onScroll={onScroll}
      role="log"
      aria-label="Log output"
      className="rounded-card mx-screen h-[60vh] min-h-64 overflow-auto bg-zinc-950 font-mono text-xs"
    >
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

function LogsStatusBar({
  entryCount,
  isPaused,
  autoScroll,
}: {
  entryCount: number
  isPaused: boolean
  autoScroll: boolean
}) {
  return (
    <div className="text-muted-foreground px-screen text-caption mt-2 flex items-center gap-4">
      <span className="nums">{entryCount} entries</span>
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {isPaused && <span className="text-yellow-500">Streaming paused</span>}
      {autoScroll ? null : <span>Auto-scroll disabled</span>}
    </div>
  )
}

function LogsControls() {
  const store = useLogsStore()
  const downloadMutation = useDownloadLogFile()

  const handleDownload = async () => {
    try {
      await downloadMutation.mutateAsync()
      toast.success('Log file downloaded')
    } catch {
      toast.error('Failed to download log file')
    }
  }

  return (
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
      onDownload={() => void handleDownload()}
      downloadPending={downloadMutation.isPending}
    />
  )
}

export function LogsPage() {
  const back = usePushBack()
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { isLoading: queryLoading } = useLogs()

  const store = useLogsStore()
  const filteredEntries = store.getFilteredEntries()
  const allSelected = store.filterLevels.length === ALL_LOG_LEVELS.length
  const { scrollRef, handleScroll } = useLogScroll(filteredEntries)
  const emptyMessage =
    store.searchText || !allSelected ? 'No logs match your filters' : 'No logs yet'

  if (queryLoading || globalLoading) {
    return (
      <Screen title="Logs" back={back}>
        <Skeleton className="mx-screen rounded-card h-[60vh] min-h-64" />
      </Screen>
    )
  }

  return (
    <Screen title="Logs" back={back}>
      <LogsControls />
      <LogScrollPanel
        entries={filteredEntries}
        emptyMessage={emptyMessage}
        scrollRef={scrollRef}
        onScroll={handleScroll}
      />
      <LogsStatusBar
        entryCount={filteredEntries.length}
        isPaused={store.isPaused}
        autoScroll={store.autoScroll}
      />
    </Screen>
  )
}
