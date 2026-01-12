import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { StatusBadge } from '@/components/media/StatusBadge'
import { MovieAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { Checkbox } from '@/components/ui/checkbox'
import type { Movie } from '@/types'

interface MovieCardProps {
  movie: Movie
  className?: string
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

export function MovieCard({ movie, className, editMode, selected, onToggleSelect }: MovieCardProps) {
  const cardContent = (
    <div className="relative aspect-[2/3]">
      <PosterImage
        tmdbId={movie.tmdbId}
        alt={movie.title}
        type="movie"
        className="absolute inset-0"
      />
      {editMode && (
        <div
          className="absolute top-2 left-2 z-10"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            onToggleSelect?.(movie.id)
          }}
        >
          <Checkbox
            checked={selected}
            className="size-5 bg-background/80 border-2"
          />
        </div>
      )}
      <div className="absolute top-2 right-2 flex flex-col gap-1 items-end">
        <StatusBadge status={movie.status} />
        <MovieAvailabilityBadge movie={movie} />
      </div>
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-3">
        <h3 className="font-semibold text-white truncate">{movie.title}</h3>
        <p className="text-sm text-gray-300">{movie.year || 'Unknown year'}</p>
      </div>
    </div>
  )

  if (editMode) {
    return (
      <div
        className={cn(
          'group block rounded-lg overflow-hidden bg-card border-2 transition-all cursor-pointer',
          selected ? 'border-primary ring-2 ring-primary/30' : 'border-border hover:border-primary/50',
          className
        )}
        onClick={() => onToggleSelect?.(movie.id)}
      >
        {cardContent}
      </div>
    )
  }

  return (
    <Link
      to="/movies/$id"
      params={{ id: String(movie.id) }}
      className={cn(
        'group block rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-primary/50 hover:shadow-lg',
        className
      )}
    >
      {cardContent}
    </Link>
  )
}
