import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'

import { SeriesListLayout } from './series-list-layout'
import { useSeriesList } from './use-series-list'

export function SeriesListPage() {
  const state = useSeriesList()

  if (state.isError) {
    return (
      <div>
        <PageHeader title="Series" />
        <ErrorState onRetry={state.refetch} />
      </div>
    )
  }

  return <SeriesListLayout state={state} />
}
