import { type ComponentType, lazy, type LazyExoticComponent, Suspense, useState } from 'react'

import { ErrorBoundary } from '@/components/error-boundary'
import { cn } from '@/lib/utils'

import { LoadingScreen } from './loading-screen'
import { paneBehindPathname, paneFromPathname, type PaneId } from './use-tab-nav'

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

const SCREEN_PANES: Partial<Record<PaneId, true>> = {
  activity: true,
  dashboard: true,
  more: true,
  movies: true,
  search: true,
  series: true,
}

const TAB_STACK_INSET = { paddingBottom: 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)' }

function PaneFrame({
  id,
  shown,
  interactive,
  Page,
}: {
  id: PaneId
  shown: boolean
  interactive: boolean
  Page: LazyExoticComponent<ComponentType>
}) {
  const screenPane = SCREEN_PANES[id] === true
  return (
    <div
      inert={!interactive}
      className={cn('absolute inset-0', screenPane ? 'overflow-hidden' : 'scroll p-6', !shown && 'invisible')}
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

export function TabPanes({ pathname, overlay }: { pathname: string; overlay: boolean }) {
  const routePane = paneFromPathname(pathname)
  const shownPane = routePane ?? (overlay ? paneBehindPathname(pathname) : null)
  const [visited, setVisited] = useState<Partial<Record<PaneId, true>>>(() =>
    shownPane === null ? {} : { [shownPane]: true },
  )

  if (shownPane !== null && visited[shownPane] !== true) {
    setVisited({ ...visited, [shownPane]: true })
  }

  return (
    <>
      {PANE_ORDER.map((id) =>
        visited[id] === true ? (
          <PaneFrame
            key={id}
            id={id}
            shown={shownPane === id}
            interactive={routePane === id}
            Page={PANE_PAGES[id]}
          />
        ) : null,
      )}
    </>
  )
}
