import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { ProductionStatusBadge } from '@/components/media/ProductionStatusBadge'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import type { Series } from '@/types'

interface SeriesCardProps {
  series: Series
  className?: string
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

export function SeriesCard({ series, className, editMode, selected, onToggleSelect }: SeriesCardProps) {
  const cardContent = (
    <div className="relative aspect-[2/3]">
      <PosterImage
        tmdbId={series.tmdbId}
        tvdbId={series.tvdbId}
        alt={series.title}
        type="series"
        version={series.updatedAt}
        className="absolute inset-0"
      />
      {editMode && (
        <div
          className="absolute top-2 left-2 z-10"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            onToggleSelect?.(series.id)
          }}
        >
          <Checkbox
            checked={selected}
            className={cn(
              'size-5 bg-background/80 border-2',
              selected && 'border-tv-500 data-[checked]:bg-tv-500'
            )}
          />
        </div>
      )}
      <div className="absolute top-2 right-2">
        <ProductionStatusBadge status={series.productionStatus} />
      </div>
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-3 pt-8">
        <h3 className="font-semibold text-white line-clamp-2 drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
          {series.title}
        </h3>
        <div className="flex items-center gap-2 text-sm text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">
          <span>{series.year || 'Unknown year'}</span>
          <Badge variant="secondary" className="text-xs">
            {(series.statusCounts.available + series.statusCounts.upgradable)}/{series.statusCounts.total} eps
          </Badge>
        </div>
      </div>
    </div>
  )

  if (editMode) {
    return (
      <div
        className={cn(
          'group block rounded-lg overflow-hidden bg-card border-2 transition-all cursor-pointer',
          selected ? 'border-tv-500 glow-tv' : 'border-border hover:border-tv-500/50 hover:glow-tv-sm',
          className
        )}
        onClick={() => onToggleSelect?.(series.id)}
      >
        {cardContent}
      </div>
    )
  }

  return (
    <Link
      to="/series/$id"
      params={{ id: String(series.id) }}
      className={cn(
        'group block rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-tv-500/50 hover:glow-tv',
        className
      )}
    >
      {cardContent}
    </Link>
  )
}
