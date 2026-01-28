import { useEffect, useRef } from 'react'
import {
  Download,
  Pause,
  Play,
  Search,
  Trash2,
  ArrowDownToLine,
  ArrowUpFromLine,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { LoadingState } from '@/components/data/LoadingState'
import { useLogs, useDownloadLogFile } from '@/hooks/useLogs'
import { useLogsStore } from '@/stores/logs'
import { cn } from '@/lib/utils'
import type { LogLevel } from '@/types/logs'

const LEVEL_COLORS: Record<string, string> = {
  trace: 'text-gray-400',
  debug: 'text-blue-400',
  info: 'text-green-400',
  warn: 'text-yellow-400',
  error: 'text-red-400',
  fatal: 'text-red-600 font-bold',
}

const LEVEL_OPTIONS: { value: LogLevel | 'all'; label: string }[] = [
  { value: 'all', label: 'All Levels' },
  { value: 'trace', label: 'Trace' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warning' },
  { value: 'error', label: 'Error' },
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
  const { isLoading } = useLogs()
  const downloadMutation = useDownloadLogFile()
  const scrollRef = useRef<HTMLDivElement>(null)

  const {
    filterLevel,
    searchText,
    isPaused,
    autoScroll,
    setFilterLevel,
    setSearchText,
    togglePaused,
    toggleAutoScroll,
    clear,
    getFilteredEntries,
  } = useLogsStore()

  const filteredEntries = getFilteredEntries()

  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [filteredEntries.length, autoScroll])

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

        <Select
          value={filterLevel}
          onValueChange={(v) => setFilterLevel(v as LogLevel | 'all')}
        >
          <SelectTrigger className="w-[140px]">
            {LEVEL_OPTIONS.find((o) => o.value === filterLevel)?.label}
          </SelectTrigger>
          <SelectContent>
            {LEVEL_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="flex-1" />

        <Button
          variant="outline"
          size="sm"
          onClick={toggleAutoScroll}
          title={autoScroll ? 'Disable auto-scroll' : 'Enable auto-scroll'}
        >
          {autoScroll ? (
            <ArrowDownToLine className="size-4" />
          ) : (
            <ArrowUpFromLine className="size-4" />
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
        className="flex-1 overflow-auto bg-zinc-950 rounded-md border font-mono text-xs"
      >
        {filteredEntries.length === 0 ? (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            {searchText || filterLevel !== 'all'
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
