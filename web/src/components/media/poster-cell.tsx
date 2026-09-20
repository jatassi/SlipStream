import { Link } from '@tanstack/react-router'
import { Check } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { StatusDot } from '@/components/media/status-dot'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import type { MediaStatus } from './media-status'

export type PosterCellItem = {
  id: number
  title: string
  href: string
  status: MediaStatus
  year?: number | null
  quality?: string | null
  posterType: 'movie' | 'series'
  tmdbId?: number
  tvdbId?: number
  version?: string | null
}

type PosterCellProps = {
  item: PosterCellItem
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

const CELL_CLASSES =
  'press focus-visible:ring-ring relative block w-full rounded-[12px] text-left outline-none focus-visible:ring-[3px]'

function CellBody({ item }: { item: PosterCellItem }) {
  return (
    <>
      <PosterImage
        tmdbId={item.tmdbId}
        tvdbId={item.tvdbId}
        alt={item.title}
        type={item.posterType}
        version={item.version}
        className="aspect-[2/3] w-full rounded-[10px] shadow-[0_6px_16px_rgba(0,0,0,0.35)]"
      />
      <div className="text-footnote mt-2 truncate font-semibold">{item.title}</div>
      <div className="text-caption text-muted-foreground mt-0.5 flex items-center gap-1.5">
        <StatusDot status={item.status} />
        <span className="nums">{item.year ?? 'Unknown'}</span>
        {typeof item.quality === 'string' && item.quality.length > 0 && (
          <>
            <span aria-hidden="true">·</span>
            <span className="truncate">{item.quality}</span>
          </>
        )}
      </div>
    </>
  )
}

function SelectionMark({ selected }: { selected: boolean }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        'absolute top-2 left-2 flex size-5 items-center justify-center rounded-md border-2',
        selected ? 'bg-primary border-primary text-background' : 'bg-background/80 border-white/70',
      )}
    >
      {selected ? <Check className="size-3.5" strokeWidth={3} /> : null}
    </span>
  )
}

export function PosterCell({ item, editMode, selected, onToggleSelect }: PosterCellProps) {
  if (editMode === true) {
    return (
      <button
        type="button"
        aria-label={`Select ${item.title}`}
        aria-pressed={selected === true}
        onClick={() => {
          onToggleSelect?.(item.id)
        }}
        className={CELL_CLASSES}
      >
        <CellBody item={item} />
        <SelectionMark selected={selected === true} />
      </button>
    )
  }

  return (
    <Link to={item.href} className={CELL_CLASSES}>
      <CellBody item={item} />
    </Link>
  )
}

export function PosterCellSkeleton() {
  return (
    <div>
      <Skeleton className="aspect-[2/3] w-full rounded-[10px]" />
      <Skeleton className="mt-2 h-4 w-4/5" />
      <Skeleton className="mt-1.5 h-3 w-3/5" />
    </div>
  )
}
