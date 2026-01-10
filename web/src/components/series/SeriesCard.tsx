import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { StatusBadge } from '@/components/media/StatusBadge'
import { SeriesAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { Badge } from '@/components/ui/badge'
import type { Series } from '@/types'

interface SeriesCardProps {
  series: Series
  className?: string
}

export function SeriesCard({ series, className }: SeriesCardProps) {
  return (
    <Link
      to="/series/$id"
      params={{ id: String(series.id) }}
      className={cn(
        'group block rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-primary/50 hover:shadow-lg',
        className
      )}
    >
      <div className="relative aspect-[2/3]">
        <PosterImage
          tmdbId={series.tmdbId}
          alt={series.title}
          type="series"
          className="absolute inset-0"
        />
        <div className="absolute top-2 right-2 flex flex-col gap-1 items-end">
          <StatusBadge status={series.status} />
          <SeriesAvailabilityBadge series={series} />
        </div>
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-3">
          <h3 className="font-semibold text-white truncate">{series.title}</h3>
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <span>{series.year || 'Unknown year'}</span>
            <Badge variant="secondary" className="text-xs">
              {series.episodeFileCount}/{series.episodeCount} eps
            </Badge>
          </div>
        </div>
      </div>
    </Link>
  )
}
