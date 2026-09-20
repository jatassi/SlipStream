import { ErrorState } from '@/components/data/error-state'
import { Screen } from '@/components/screen/screen'

import { SeriesListLayout } from './series-list-layout'
import { useSeriesList } from './use-series-list'

export function SeriesListPage() {
  const state = useSeriesList()

  if (state.isError) {
    return (
      <Screen title="Series">
        <ErrorState onRetry={state.refetch} />
      </Screen>
    )
  }

  return <SeriesListLayout state={state} />
}
