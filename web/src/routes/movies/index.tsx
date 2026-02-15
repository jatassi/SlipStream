import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'

import { MovieListLayout } from './movie-list-layout'
import { useMovieList } from './use-movie-list'

export function MoviesPage() {
  const state = useMovieList()

  if (state.isError) {
    return (
      <div>
        <PageHeader title="Movies" />
        <ErrorState onRetry={state.refetch} />
      </div>
    )
  }

  return <MovieListLayout state={state} />
}
