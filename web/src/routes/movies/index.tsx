import { ErrorState } from '@/components/data/error-state'
import { Screen } from '@/components/screen/screen'

import { MovieListLayout } from './movie-list-layout'
import { useMovieList } from './use-movie-list'

export function MoviesPage() {
  const state = useMovieList()

  if (state.isError) {
    return (
      <Screen title="Movies">
        <ErrorState onRetry={state.refetch} />
      </Screen>
    )
  }

  return <MovieListLayout state={state} />
}
