import { useRef, useState } from 'react'

import { Ellipsis, Pause, Play } from 'lucide-react'

import { ProgressLine } from '@/components/grouped-list'
import { RowThumbnail } from '@/components/media/row-thumbnail'
import { ActionPresenter } from '@/components/presenter'
import { useViewport } from '@/hooks/use-viewport'
import type { QueueItem } from '@/types'

import { useQueueRow } from './use-queue-row'

type Row = ReturnType<typeof useQueueRow>

const CONTROL =
  'press size-tap focus-visible:ring-ring flex shrink-0 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]'

function QueueThumbnail({ item, row }: { item: QueueItem; row: Row }) {
  return (
    <RowThumbnail
      tmdbId={row.tmdbId}
      tvdbId={row.tvdbId}
      alt={item.title}
      type={item.mediaType === 'movie' ? 'movie' : 'series'}
      size="md"
    />
  )
}

function RowBody({ item, row, onOpen }: { item: QueueItem; row: Row; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="focus-visible:ring-ring flex min-w-0 flex-1 items-center gap-3 text-left outline-none focus-visible:ring-[3px]"
    >
      <QueueThumbnail item={item} row={row} />
      <span className="min-w-0 flex-1">
        <span className="flex items-baseline gap-2">
          <span className="text-body truncate font-medium">{item.title}</span>
          {row.episodeLabel !== '' && (
            <span className="nums text-caption text-muted-foreground shrink-0">
              {row.episodeLabel}
            </span>
          )}
        </span>
        <span className="text-muted-foreground/80 mt-0.5 block truncate font-mono text-[11px]">
          {item.releaseName}
        </span>
        <ProgressLine
          value={item.progress}
          kind={item.mediaType === 'movie' ? 'movie' : 'series'}
          muted={row.isPaused}
          className="mt-2"
        />
        <span className="nums text-caption text-muted-foreground mt-1.5 block">
          {row.statsLine}
        </span>
      </span>
    </button>
  )
}

function TrailingControl({ item, row }: { item: QueueItem; row: Row }) {
  if (!row.canToggle) {
    return (
      <span className="text-footnote text-muted-foreground shrink-0 px-2">{row.stateLabel}</span>
    )
  }
  const Icon = row.isPaused ? Play : Pause
  return (
    <button
      type="button"
      onClick={row.toggle}
      disabled={row.togglePending}
      aria-label={`${row.isPaused ? 'Resume' : 'Pause'} ${item.title}`}
      className={`${CONTROL} text-foreground`}
    >
      <Icon className="size-5 fill-current" />
    </button>
  )
}

function MoreButton({ title, onOpen }: { title: string; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`More actions for ${title}`}
      className={`${CONTROL} text-muted-foreground`}
    >
      <Ellipsis className="size-5" />
    </button>
  )
}

export function QueueRow({ item }: { item: QueueItem }) {
  const row = useQueueRow(item)
  const [open, setOpen] = useState(false)
  const anchor = useRef<HTMLDivElement>(null)
  const shell = useViewport()
  const onOpen = () => {
    setOpen(true)
  }

  return (
    <div ref={anchor} className="press-row flex items-center gap-3 px-4 py-3">
      <RowBody item={item} row={row} onOpen={onOpen} />
      <TrailingControl item={item} row={row} />
      {shell === 'wide' && <MoreButton title={item.title} onOpen={onOpen} />}
      <ActionPresenter
        open={open}
        onOpenChange={setOpen}
        title={item.title}
        subtitle={item.releaseName}
        actions={row.actions}
        anchor={anchor}
      />
    </div>
  )
}
