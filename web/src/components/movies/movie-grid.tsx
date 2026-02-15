import type { Movie } from '@/types'

import { MovieCard } from './movie-card'

type MovieGridProps = {
  movies: Movie[]
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function MovieGrid({
  movies,
  posterSize = 150,
  editMode,
  selectedIds,
  onToggleSelect,
}: MovieGridProps) {
  return (
    <div
      className="grid gap-4"
      style={{
        gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))`,
      }}
    >
      {movies.map((movie) => (
        <MovieCard
          key={movie.id}
          movie={movie}
          editMode={editMode}
          selected={selectedIds?.has(movie.id)}
          onToggleSelect={onToggleSelect}
        />
      ))}
    </div>
  )
}
