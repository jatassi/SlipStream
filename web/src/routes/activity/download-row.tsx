import type { RefObject } from 'react'
import { useContext, useEffect, useRef } from 'react'

import { cn } from '@/lib/utils'
import type { QueueItem } from '@/types'

import { DownloadRowActions } from './download-row-actions'
import { DownloadRowPoster } from './download-row-poster'
import { DownloadRowProgress } from './download-row-progress'
import { TitleWidthContext } from './title-width-context'
import { useDownloadRow } from './use-download-row'

function useTitleWidth(rowId: string, showReleaseName: boolean) {
  const titleRef = useRef<HTMLDivElement>(null)
  const { registerWidth, unregisterWidth, maxWidth } = useContext(TitleWidthContext)

  useEffect(() => {
    const measure = () => {
      if (titleRef.current) {
        registerWidth(rowId, titleRef.current.scrollWidth)
      }
    }
    measure()
    const timer = setTimeout(measure, 0)
    return () => {
      clearTimeout(timer)
      unregisterWidth(rowId)
    }
  }, [rowId, showReleaseName, registerWidth, unregisterWidth])

  return { titleRef, maxWidth }
}

type TitleCellProps = {
  titleRef: RefObject<HTMLDivElement | null>
  maxWidth: number
  isMovie: boolean
  isSeries: boolean
  title: string
  titleSuffix: string
  releaseName: string
  showReleaseName: boolean
  onToggleReleaseName: () => void
}

function TitleCell({
  titleRef,
  maxWidth,
  isMovie,
  isSeries,
  title,
  titleSuffix,
  releaseName,
  showReleaseName,
  onToggleReleaseName,
}: TitleCellProps) {
  return (
    <div
      className="shrink-0 self-center overflow-hidden transition-[width] duration-150 ease-out"
      style={{ width: maxWidth > 0 ? maxWidth : 'auto' }}
    >
      <div ref={titleRef} className="inline-block">
        <button
          type="button"
          className={cn(
            'cursor-pointer font-medium whitespace-nowrap transition-colors',
            isMovie && 'hover:text-movie-500',
            isSeries && 'hover:text-tv-500',
          )}
          title={releaseName}
          onClick={onToggleReleaseName}
        >
          {title}
          {titleSuffix ? (
            <span className="text-muted-foreground ml-1.5">{titleSuffix}</span>
          ) : null}
        </button>
        {showReleaseName ? (
          <div className="text-muted-foreground mt-0.5 animate-[slide-down-fade_150ms_ease-out] text-xs whitespace-nowrap">
            {releaseName}
          </div>
        ) : null}
      </div>
    </div>
  )
}

type DownloadRowProps = {
  item: QueueItem
  showReleaseName: boolean
  onToggleReleaseName: () => void
}

export function DownloadRow({ item, showReleaseName, onToggleReleaseName }: DownloadRowProps) {
  const row = useDownloadRow(item)
  const { titleRef, maxWidth } = useTitleWidth(`${item.clientId}-${item.id}`, showReleaseName)

  return (
    <div
      className={cn(
        'flex items-center gap-4 px-4 py-3 transition-colors',
        row.isMovie && 'hover:bg-movie-500/5',
        row.isSeries && 'hover:bg-tv-500/5',
        !row.isMovie && !row.isSeries && 'hover:bg-accent/50',
      )}
    >
      <div className="shrink-0 self-center">
        <DownloadRowPoster mediaType={item.mediaType} tmdbId={row.tmdbId} tvdbId={row.tvdbId} alt={item.title} />
      </div>

      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-0.5">
        <TitleCell
          titleRef={titleRef}
          maxWidth={maxWidth}
          isMovie={row.isMovie}
          isSeries={row.isSeries}
          title={item.title}
          titleSuffix={row.titleSuffix}
          releaseName={item.releaseName}
          showReleaseName={showReleaseName}
          onToggleReleaseName={onToggleReleaseName}
        />
        <DownloadRowProgress item={item} isMovie={row.isMovie} isSeries={row.isSeries} progressText={row.progressText} />
      </div>

      <DownloadRowActions
        item={item}
        isMovie={row.isMovie}
        isSeries={row.isSeries}
        pauseIsPending={row.pauseIsPending}
        resumeIsPending={row.resumeIsPending}
        fastForwardIsPending={row.fastForwardIsPending}
        onPause={row.handlePause}
        onResume={row.handleResume}
        onFastForward={row.handleFastForward}
        onRemove={row.handleRemove}
      />
    </div>
  )
}
