import { Link } from '@tanstack/react-router'

import { MediaStatusBadge } from '@/components/media/media-status-badge'
import { PosterImage } from '@/components/media/poster-image'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import type { Movie } from '@/types'

type MovieCardProps = {
  movie: Movie
  className?: string
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

export function MovieCard({
  movie,
  className,
  editMode,
  selected,
  onToggleSelect,
}: MovieCardProps) {
  const cardContent = (
    <MovieCardContent
      movie={movie}
      editMode={editMode}
      selected={selected}
      onToggleSelect={onToggleSelect}
    />
  )

  if (editMode) {
    return (
      <button
        type="button"
        className={cn(
          'group bg-card block cursor-pointer overflow-hidden rounded-lg border-2 transition-all w-full',
          selected
            ? 'border-movie-500 glow-movie'
            : 'border-border hover:border-movie-500/50 hover:glow-movie-sm',
          className,
        )}
        onClick={() => onToggleSelect?.(movie.id)}
      >
        {cardContent}
      </button>
    )
  }

  return (
    <Link
      to="/movies/$id"
      params={{ id: String(movie.id) }}
      className={cn(
        'group bg-card border-border hover:border-movie-500/50 hover:glow-movie block overflow-hidden rounded-lg border transition-all',
        className,
      )}
    >
      {cardContent}
    </Link>
  )
}

function MovieCardContent({
  movie,
  editMode,
  selected,
  onToggleSelect,
}: {
  movie: Movie
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}) {
  return (
    <div className="relative aspect-[2/3]">
      <PosterImage
        tmdbId={movie.tmdbId}
        alt={movie.title}
        type="movie"
        version={movie.updatedAt}
        className="absolute inset-0"
      />
      {editMode ? (
        <MovieEditCheckbox movieId={movie.id} selected={selected} onToggle={onToggleSelect} />
      ) : null}
      <div className="absolute top-2 right-2">
        <MediaStatusBadge status={movie.status} />
      </div>
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-3 pt-8">
        <h3 className="line-clamp-2 font-semibold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
          {movie.title}
        </h3>
        <p className="text-sm text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">
          {movie.year ?? 'Unknown year'}
        </p>
      </div>
    </div>
  )
}

function MovieEditCheckbox({
  movieId,
  selected,
  onToggle,
}: {
  movieId: number
  selected?: boolean
  onToggle?: (id: number) => void
}) {
  return (
    <button
      type="button"
      className="absolute top-2 left-2 z-10"
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        onToggle?.(movieId)
      }}
    >
      <Checkbox
        checked={selected}
        className={cn(
          'bg-background/80 size-5 border-2',
          selected && 'border-movie-500 data-[checked]:bg-movie-500',
        )}
      />
    </button>
  )
}
