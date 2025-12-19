import { cn } from '@/lib/utils'
import { MovieCard } from './MovieCard'
import type { Movie } from '@/types'

interface MovieGridProps {
  movies: Movie[]
  className?: string
}

export function MovieGrid({ movies, className }: MovieGridProps) {
  return (
    <div
      className={cn(
        'grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6',
        className
      )}
    >
      {movies.map((movie) => (
        <MovieCard key={movie.id} movie={movie} />
      ))}
    </div>
  )
}
