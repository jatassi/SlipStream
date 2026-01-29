import { useState, useMemo } from 'react'
import { Download } from 'lucide-react'
import { Progress } from '@/components/ui/progress'
import { usePortalDownloads } from '@/hooks'
import { formatEta, formatSpeed } from '@/lib/formatters'
import type { PortalDownload } from '@/types'

function DownloadRow({ download }: { download: PortalDownload }) {
  const isActive = download.status === 'downloading'
  const isPaused = download.status === 'paused'
  const isComplete = Math.round(download.progress) >= 100

  const title = download.requestTitle || download.title
  const season = download.mediaType === 'series' && download.seasonNumber != null
    ? `S${download.seasonNumber}`
    : null
  const fullTitle = season ? `${title} ${season}` : title

  return (
    <div className="flex items-center gap-1.5 md:gap-2 sm:gap-3 px-2 md:px-3 sm:px-4 py-1.5 md:py-2 border-b border-border last:border-b-0 bg-card/50">
      <Download className="size-3 md:size-4 text-primary shrink-0" />
      <span className="text-xs md:text-sm font-medium truncate min-w-0 flex-1" title={fullTitle}>
        {title}
        {season && <span className="text-muted-foreground font-normal"> {season}</span>}
      </span>
      <div className="flex items-center gap-1 sm:gap-3 shrink-0">
        <Progress value={download.progress} className="w-16 md:w-20 sm:w-48 h-1 md:h-1.5" />
        <span className="text-[10px] md:text-xs text-muted-foreground w-7 md:w-8 sm:w-10 text-right">
          {Math.round(download.progress)}%
        </span>
        {isActive && download.downloadSpeed > 0 && !isComplete && (
          <span className="hidden sm:inline text-xs text-muted-foreground w-20 text-right">
            {formatSpeed(download.downloadSpeed)}
          </span>
        )}
        {isComplete && (
          <span className="hidden sm:inline text-xs text-muted-foreground w-20 text-right">--</span>
        )}
        <span className="text-[10px] md:text-xs text-muted-foreground w-12 md:w-14 sm:w-16 text-right">
          {isComplete ? 'Importing' : isPaused ? 'Paused' : isActive ? formatEta(download.eta) : download.status}
        </span>
      </div>
    </div>
  )
}

export function PortalDownloads() {
  const [seenOrder, setSeenOrder] = useState<string[]>([])

  const { data: downloads } = usePortalDownloads()

  const activeDownloads = downloads?.filter(
    d => d.status === 'downloading' || d.status === 'queued' || d.status === 'paused'
  ) || []

  // Maintain stable order: keep items in the order they were first seen
  const activeKeys = new Set(activeDownloads.map(d => `${d.clientId}-${d.id}`))

  // Compute the new order based on current active keys
  const newOrder = useMemo(() => {
    // Keep existing keys that are still active
    const kept = seenOrder.filter(key => activeKeys.has(key))
    // Add new keys at the end
    const newKeys = activeDownloads
      .map(d => `${d.clientId}-${d.id}`)
      .filter(key => !kept.includes(key))
    return [...kept, ...newKeys]
  }, [seenOrder, activeKeys, activeDownloads])

  // Update state if order changed (React-recommended render-time adjustment pattern)
  if (newOrder.length !== seenOrder.length || newOrder.some((key, i) => key !== seenOrder[i])) {
    setSeenOrder(newOrder)
  }

  // Sort by insertion order
  const downloadMap = new Map(activeDownloads.map(d => [`${d.clientId}-${d.id}`, d]))
  const sortedDownloads = newOrder
    .map(key => downloadMap.get(key))
    .filter((d): d is PortalDownload => d !== undefined)

  if (sortedDownloads.length === 0) {
    return null
  }

  return (
    <div className="border-b border-border bg-muted/30">
      {sortedDownloads.map(download => (
        <DownloadRow key={`${download.clientId}-${download.id}`} download={download} />
      ))}
    </div>
  )
}
