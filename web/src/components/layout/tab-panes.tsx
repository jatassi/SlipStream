import { type ComponentType, lazy, type LazyExoticComponent, Suspense, useState } from 'react'

import { ErrorBoundary } from '@/components/error-boundary'
import { cn } from '@/lib/utils'

import { LoadingScreen } from './loading-screen'
import { paneFromPathname,type PaneId } from './use-tab-nav'

const DashboardPage = lazy(() => import('@/routes/index').then((m) => ({ default: m.DashboardPage })))
const MoviesPage = lazy(() => import('@/routes/movies/index').then((m) => ({ default: m.MoviesPage })))
const SeriesListPage = lazy(() =>
  import('@/routes/series/index').then((m) => ({ default: m.SeriesListPage })),
)
const ActivityPage = lazy(() =>
  import('@/routes/downloads/index').then((m) => ({ default: m.ActivityPage })),
)
const SearchPage = lazy(() => import('@/routes/search/index').then((m) => ({ default: m.SearchPage })))
const MorePage = lazy(() => import('@/routes/more/index').then((m) => ({ default: m.MorePage })))

const PANE_PAGES: Record<PaneId, LazyExoticComponent<ComponentType>> = {
  dashboard: DashboardPage,
  movies: MoviesPage,
  series: SeriesListPage,
  activity: ActivityPage,
  search: SearchPage,
  more: MorePage,
}

const PANE_ORDER: PaneId[] = ['dashboard', 'movies', 'series', 'activity', 'search', 'more']

const SCREEN_PANES: Partial<Record<PaneId, true>> = { dashboard: true, more: true }

const TAB_STACK_INSET = { paddingBottom: 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)' }

function PaneFrame({
  id,
  active,
  Page,
}: {
  id: PaneId
  active: boolean
  Page: LazyExoticComponent<ComponentType>
}) {
  const screenPane = SCREEN_PANES[id] === true
  return (
    <div
      inert={!active}
      className={cn('absolute inset-0', screenPane ? 'overflow-hidden' : 'scroll p-6', !active && 'invisible')}
      style={screenPane ? undefined : TAB_STACK_INSET}
    >
      <ErrorBoundary>
        <Suspense fallback={<LoadingScreen />}>
          <Page />
        </Suspense>
      </ErrorBoundary>
    </div>
  )
}

export function TabPanes({ pathname }: { pathname: string }) {
  const pane = paneFromPathname(pathname)
  const [visited, setVisited] = useState<Partial<Record<PaneId, true>>>(() =>
    pane === null ? {} : { [pane]: true },
  )

  if (pane !== null && visited[pane] !== true) {
    setVisited({ ...visited, [pane]: true })
  }

  return (
    <>
      {PANE_ORDER.map((id) =>
        visited[id] === true ? (
          <PaneFrame key={id} id={id} active={pane === id} Page={PANE_PAGES[id]} />
        ) : null,
      )}
    </>
  )
}
