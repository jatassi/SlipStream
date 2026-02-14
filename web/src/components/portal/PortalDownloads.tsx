import { useMemo, useState } from 'react'

import { Download } from 'lucide-react'

import { Progress } from '@/components/ui/progress'
import { usePortalDownloads } from '@/hooks'
import { formatEta, formatSpeed } from '@/lib/formatters'
import type { PortalDownload } from '@/types'

function DownloadRow({ download }: { download: PortalDownload }) {
  const isActive = download.status === 'downloading'
  const isPaused = download.status === 'paused'
  const isComplete = Math.round(download.progress) >= 100
  const isMovie = download.mediaType === 'movie'

  const title = download.requestTitle || download.title
  const season =
    download.mediaType === 'series' && download.seasonNumber != null
      ? `S${download.seasonNumber}`
      : null
  const fullTitle = season ? `${title} ${season}` : title

  return (
    <div
      className={`flex items-center gap-1.5 border-b px-2 py-1.5 last:border-b-0 sm:gap-3 sm:px-4 md:gap-2 md:px-3 md:py-2 ${
        isMovie ? 'border-movie-500/20 bg-movie-500/5' : 'border-tv-500/20 bg-tv-500/5'
      }`}
    >
      <Download
        className={`size-3 shrink-0 md:size-4 ${isMovie ? 'text-movie-400' : 'text-tv-400'}`}
      />
      <span className="min-w-0 flex-1 truncate text-xs font-medium md:text-sm" title={fullTitle}>
        {title}
        {season ? <span className="text-muted-foreground font-normal"> {season}</span> : null}
      </span>
      <div className="flex shrink-0 items-center gap-1 sm:gap-3">
        <Progress
          value={download.progress}
          variant={isMovie ? 'movie' : 'tv'}
          className="h-1 w-16 sm:w-48 md:h-1.5 md:w-20"
        />
        <span className="text-muted-foreground w-7 text-right text-[10px] sm:w-10 md:w-8 md:text-xs">
          {Math.round(download.progress)}%
        </span>
        {isActive && download.downloadSpeed > 0 && !isComplete ? (
          <span className="text-muted-foreground hidden w-20 text-right text-xs sm:inline">
            {formatSpeed(download.downloadSpeed)}
          </span>
        ) : null}
        {isComplete ? (
          <span className="text-muted-foreground hidden w-20 text-right text-xs sm:inline">--</span>
        ) : null}
        <span className="text-muted-foreground w-12 text-right text-[10px] sm:w-16 md:w-14 md:text-xs">
          {isComplete
            ? 'Importing'
            : isPaused
              ? 'Paused'
              : isActive
                ? formatEta(download.eta)
                : download.status}
        </span>
      </div>
    </div>
  )
}

export function PortalDownloads() {
  const [seenOrder, setSeenOrder] = useState<string[]>([])

  const { data: downloads } = usePortalDownloads()

  const activeDownloads = useMemo(
    () =>
      downloads?.filter(
        (d) => d.status === 'downloading' || d.status === 'queued' || d.status === 'paused',
      ) ?? [],
    [downloads],
  )

  // Maintain stable order: keep items in the order they were first seen
  const activeKeys = useMemo(
    () => new Set(activeDownloads.map((d) => `${d.clientId}-${d.id}`)),
    [activeDownloads],
  )

  // Compute the new order based on current active keys
  const newOrder = useMemo(() => {
    // Keep existing keys that are still active
    const kept = seenOrder.filter((key) => activeKeys.has(key))
    // Add new keys at the end
    const newKeys = activeDownloads
      .map((d) => `${d.clientId}-${d.id}`)
      .filter((key) => !kept.includes(key))
    return [...kept, ...newKeys]
  }, [seenOrder, activeKeys, activeDownloads])

  // Update state if order changed (React-recommended render-time adjustment pattern)
  if (newOrder.length !== seenOrder.length || newOrder.some((key, i) => key !== seenOrder[i])) {
    setSeenOrder(newOrder)
  }

  // Sort by insertion order
  const downloadMap = new Map(activeDownloads.map((d) => [`${d.clientId}-${d.id}`, d]))
  const sortedDownloads = newOrder
    .map((key) => downloadMap.get(key))
    .filter((d): d is PortalDownload => d !== undefined)

  if (sortedDownloads.length === 0) {
    return null
  }

  return (
    <div className="border-border bg-muted/30 border-b">
      {sortedDownloads.map((download) => (
        <DownloadRow key={`${download.clientId}-${download.id}`} download={download} />
      ))}
    </div>
  )
}
