import { EmptyState } from '@/components/data/empty-state'
import { ErrorState } from '@/components/data/error-state'
import { Group, RowSkeleton } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { MissingMovieRows, UpgradableMovieRows } from '@/components/missing/missing-movie-rows'
import { MissingSeriesRows, UpgradableSeriesRows } from '@/components/missing/missing-series-rows'
import { Screen } from '@/components/screen/screen'
import { Segmented, type SegmentedOption } from '@/components/ui/segmented'

import type { ViewMode } from './use-missing-page'
import { useMissingPage } from './use-missing-page'

const SKELETONS = ['missing-a', 'missing-b', 'missing-c', 'missing-d', 'missing-e'] as const

const VIEW_OPTIONS: SegmentedOption<ViewMode>[] = [
  { value: 'missing', label: 'Missing' },
  { value: 'upgradable', label: 'Upgradable' },
]

type PageState = ReturnType<typeof useMissingPage>

function SearchAllAction({ state }: { state: PageState }) {
  if (state.count === 0) {
    return null
  }
  return (
    <button
      type="button"
      onClick={state.handleSearchAll}
      disabled={state.isSearching}
      className="press-dim min-h-tap text-title focus-visible:ring-ring px-2 outline-none focus-visible:ring-[3px] disabled:opacity-60"
    >
      {state.isSearching ? 'Searching' : 'Search All'}
    </button>
  )
}

function unitLabel(moduleId: string, count: number): string {
  if (moduleId === 'movie') {
    return count === 1 ? 'movie' : 'movies'
  }
  return count === 1 ? 'episode' : 'episodes'
}

function emptyTitle(moduleId: string, view: ViewMode): string {
  const noun = moduleId === 'movie' ? 'movies' : 'episodes'
  return view === 'missing' ? `No missing ${noun}` : `No upgradable ${noun}`
}

function emptyDescription(view: ViewMode): string {
  return view === 'missing'
    ? 'Everything monitored and released has been downloaded'
    : 'Everything monitored meets its quality cutoff'
}

function MissingRows({ state }: { state: PageState }) {
  if (state.view === 'missing') {
    if (state.moduleId === 'movie') {
      return (
        <MissingMovieRows movies={state.missingMovies} profileNames={state.qualityProfileNames} />
      )
    }
    return <MissingSeriesRows series={state.missingSeries} />
  }
  if (state.moduleId === 'movie') {
    return (
      <UpgradableMovieRows movies={state.upgradableMovies} profiles={state.qualityProfileMap} />
    )
  }
  return <UpgradableSeriesRows series={state.upgradableSeries} />
}

function MissingContent({ state }: { state: PageState }) {
  if (state.isLoading) {
    return (
      <Group>
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} leading="poster" trailing />
        ))}
      </Group>
    )
  }

  if (state.count === 0) {
    return (
      <EmptyState
        title={emptyTitle(state.moduleId, state.view)}
        description={emptyDescription(state.view)}
      />
    )
  }

  return (
    <Group header={`${state.count} ${unitLabel(state.moduleId, state.count)}`}>
      <MissingRows state={state} />
    </Group>
  )
}

export function MissingPage() {
  const state = useMissingPage()
  const back = usePushBack()

  if (state.isError) {
    return (
      <Screen title="Missing" back={back}>
        <ErrorState onRetry={state.handleRefetch} />
      </Screen>
    )
  }

  return (
    <Screen title="Missing" back={back} trailing={<SearchAllAction state={state} />}>
      <div className="px-screen space-y-3 pb-5">
        {state.modules.length > 1 && (
          <Segmented
            label="Missing module"
            value={state.moduleId}
            onChange={state.setModuleId}
            options={state.modules.map((mod) => ({ value: mod.id, label: mod.name }))}
          />
        )}
        <Segmented
          label="Missing view"
          value={state.view}
          onChange={state.setView}
          options={VIEW_OPTIONS}
        />
      </div>
      <MissingContent state={state} />
    </Screen>
  )
}
