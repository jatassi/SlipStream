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
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { LoadingState } from '@/components/data/LoadingState'
import { useGlobalLoading } from '@/hooks'
import { useLogs, useDownloadLogFile } from '@/hooks/useLogs'
import { useLogsStore, ALL_LOG_LEVELS } from '@/stores/logs'
import { cn } from '@/lib/utils'
import type { LogLevel } from '@/types/logs'

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
  if (!fields || Object.keys(fields).length === 0) return ''
  return Object.entries(fields)
    .map(([k, v]) => `${k}=${JSON.stringify(v)}`)
    .join(' ')
}

export function LogsPage() {
  const globalLoading = useGlobalLoading()
  const { isLoading: queryLoading } = useLogs()
  const isLoading = queryLoading || globalLoading
  const downloadMutation = useDownloadLogFile()
  const scrollRef = useRef<HTMLDivElement>(null)

  const {
    filterLevels,
    searchText,
    isPaused,
    autoScroll,
    toggleFilterLevel,
    resetFilterLevels,
    setSearchText,
    togglePaused,
    toggleAutoScroll,
    setAutoScroll,
    clear,
    getFilteredEntries,
  } = useLogsStore()

  const filteredEntries = getFilteredEntries()
  const allSelected = filterLevels.length === ALL_LOG_LEVELS.length

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [filteredEntries.length, autoScroll])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
    if (atBottom && !autoScroll) {
      setAutoScroll(true)
    } else if (!atBottom && autoScroll) {
      setAutoScroll(false)
    }
  }, [autoScroll, setAutoScroll])

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
        <PageHeader
          title="System Logs"
          description="Real-time log streaming"
        />
        <LoadingState variant="list" count={10} />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-[calc(100vh-120px)]">
      <PageHeader
        title="System Logs"
        description="Real-time log streaming"
      />

      <div className="flex items-center gap-2 mb-4">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            placeholder="Search logs..."
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            className="pl-8"
          />
        </div>

        <FilterDropdown
          options={LEVEL_OPTIONS}
          selected={filterLevels}
          onToggle={toggleFilterLevel}
          onReset={resetFilterLevels}
          label="Levels"
        />

        <div className="flex-1" />

        <Button
          variant="outline"
          size="sm"
          onClick={toggleAutoScroll}
          title={autoScroll ? 'Disable auto-scroll' : 'Enable auto-scroll'}
        >
          {autoScroll ? (
            <ChevronsDown className="size-4" />
          ) : (
            <ChevronsUp className="size-4" />
          )}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={togglePaused}
          title={isPaused ? 'Resume streaming' : 'Pause streaming'}
        >
          {isPaused ? (
            <Play className="size-4" />
          ) : (
            <Pause className="size-4" />
          )}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={clear}
          title="Clear logs"
        >
          <Trash2 className="size-4" />
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handleDownload}
          disabled={downloadMutation.isPending}
          title="Download log file"
        >
          <Download className="size-4" />
        </Button>
      </div>

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto bg-zinc-950 rounded-md border font-mono text-xs"
      >
        {filteredEntries.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            {searchText || !allSelected
              ? 'No logs match your filters'
              : 'No logs yet'}
          </div>
        ) : (
          <div className="p-2 space-y-px">
            {filteredEntries.map((entry, idx) => {
              const fields = formatFields(entry.fields)
              return (
                <div
                  key={`${entry.timestamp}-${idx}`}
                  className="flex gap-2 hover:bg-zinc-900 px-1 rounded"
                >
                  <span className="text-zinc-500 shrink-0">
                    {formatTimestamp(entry.timestamp)}
                  </span>
                  <span
                    className={cn(
                      'uppercase w-12 shrink-0',
                      LEVEL_COLORS[entry.level] || 'text-zinc-400'
                    )}
                  >
                    {entry.level.slice(0, 5).padEnd(5)}
                  </span>
                  {entry.component && (
                    <span className="text-cyan-400 shrink-0">
                      [{entry.component}]
                    </span>
                  )}
                  <span className="text-zinc-100">{entry.message}</span>
                  {fields && (
                    <span className="text-zinc-500">{fields}</span>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className="mt-2 text-xs text-muted-foreground flex items-center gap-4">
        <span>{filteredEntries.length} entries</span>
        {isPaused && <span className="text-yellow-500">Streaming paused</span>}
        {!autoScroll && <span>Auto-scroll disabled</span>}
      </div>
    </div>
  )
}
