import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { PosterImage } from '@/components/media/PosterImage'
import { StatusBadge } from '@/components/media/StatusBadge'
import type { Movie } from '@/types'

interface MovieCardProps {
  movie: Movie
  className?: string
}

export function MovieCard({ movie, className }: MovieCardProps) {
  return (
    <Link
      to="/movies/$id"
      params={{ id: String(movie.id) }}
      className={cn(
        'group block rounded-lg overflow-hidden bg-card border border-border transition-all hover:border-primary/50 hover:shadow-lg',
        className
      )}
    >
      <div className="relative aspect-[2/3]">
        <PosterImage
          path={undefined} // TODO: Get from metadata
          alt={movie.title}
          type="movie"
          className="absolute inset-0"
        />
        <div className="absolute top-2 right-2">
          <StatusBadge status={movie.status} />
        </div>
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-3">
          <h3 className="font-semibold text-white truncate">{movie.title}</h3>
          <p className="text-sm text-gray-300">{movie.year || 'Unknown year'}</p>
        </div>
      </div>
    </Link>
  )
}
