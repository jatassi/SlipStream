import { createRootRoute, createRoute, lazyRouteComponent, Outlet, redirect } from '@tanstack/react-router'

import { RootLayout } from '@/components/layout/root-layout'
import { PortalAuthGuard, PortalLayout } from '@/components/portal'
import { extendedMovieMetadataOptions, extendedSeriesMetadataOptions } from '@/hooks/use-metadata'
import { movieQueryOptions } from '@/hooks/use-movies'
import { seriesQueryOptions } from '@/hooks/use-series'
import { queryClient } from '@/lib/query-client'

export const rootRoute = createRootRoute({
  component: () => (
    <RootLayout>
      <Outlet />
    </RootLayout>
  ),
})

const lazyRoute = <P extends string>(
  path: P,
  importer: () => Promise<Record<string, unknown>>,
  exportName: string,
) =>
  createRoute({
    getParentRoute: () => rootRoute,
    path,
    component: lazyRouteComponent(importer, exportName),
  })

const redirectRoute = <P extends string>(path: P, to: string) =>
  createRoute({
    getParentRoute: () => rootRoute,
    path,
    // TanStack Router requires throw redirect() — framework pattern
    beforeLoad: () => {
      // eslint-disable-next-line @typescript-eslint/only-throw-error -- TanStack Router redirect pattern
      throw redirect({ to })
    },
  })

export const authSetupRoute = lazyRoute('/auth/setup', () => import('@/routes/auth/setup'), 'SetupPage')
export const indexRoute = lazyRoute('/', () => import('@/routes/index'), 'DashboardPage')

export const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/search',
  component: lazyRouteComponent(() => import('@/routes/search/index'), 'SearchPage'),
  validateSearch: (search: Record<string, unknown>): { q: string } => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
})

export const moviesRoute = lazyRoute('/movies', () => import('@/routes/movies/index'), 'MoviesPage')

export const movieDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/movies/$id',
  component: lazyRouteComponent(() => import('@/routes/movies/$id'), 'MovieDetailPage'),
  loader: async ({ params }) => {
    const movieId = Number(params.id)
    const movie = await queryClient.ensureQueryData(movieQueryOptions(movieId))
    if (movie.tmdbId) {
      void queryClient.prefetchQuery(extendedMovieMetadataOptions(movie.tmdbId))
    }
  },
})

export const addMovieRoute = lazyRoute('/movies/add', () => import('@/routes/movies/add'), 'AddMoviePage')
export const seriesRoute = lazyRoute('/series', () => import('@/routes/series/index'), 'SeriesListPage')

export const seriesDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/$id',
  component: lazyRouteComponent(() => import('@/routes/series/$id'), 'SeriesDetailPage'),
  loader: async ({ params }) => {
    const seriesId = Number(params.id)
    const series = await queryClient.ensureQueryData(seriesQueryOptions(seriesId))
    if (series.tmdbId) {
      void queryClient.prefetchQuery(extendedSeriesMetadataOptions(series.tmdbId))
    }
  },
})
export const addSeriesRoute = lazyRoute('/series/add', () => import('@/routes/series/add'), 'AddSeriesPage')
export const calendarRoute = lazyRoute('/calendar', () => import('@/routes/calendar/index'), 'CalendarPage')
export const missingRoute = lazyRoute('/missing', () => import('@/routes/missing/index'), 'MissingPage')
export const activityRoute = lazyRoute('/downloads', () => import('@/routes/downloads/index'), 'ActivityPage')
export const historyRoute = lazyRoute('/history', () => import('@/routes/history/history'), 'HistoryPage')

// Settings — Media
export const settingsRoute = redirectRoute('/settings', '/settings/media/root-folders')
export const mediaSettingsRoute = redirectRoute('/settings/media', '/settings/media/root-folders')
export const rootFoldersRoute = lazyRoute('/settings/media/root-folders', () => import('@/routes/settings/media/root-folders'), 'RootFoldersPage')
export const qualityProfilesRoute = lazyRoute('/settings/media/quality-profiles', () => import('@/routes/settings/media/quality-profiles'), 'QualityProfilesPage')
export const versionSlotsRoute = lazyRoute('/settings/media/version-slots', () => import('@/routes/settings/media/version-slots'), 'VersionSlotsPage')
export const fileNamingRoute = lazyRoute('/settings/media/file-naming', () => import('@/routes/settings/media/file-naming'), 'FileNamingPage')
export const arrImportRoute = lazyRoute('/settings/media/arr-import', () => import('@/routes/settings/media/arr-import'), 'ArrImportPage')

// Settings — Download Pipeline
export const downloadPipelineRoute = redirectRoute('/settings/download-pipeline', '/settings/download-pipeline/indexers')
export const indexersRoute = lazyRoute('/settings/download-pipeline/indexers', () => import('@/routes/settings/download-pipeline/indexers'), 'IndexersPage')
export const downloadClientsRoute = lazyRoute('/settings/download-pipeline/clients', () => import('@/routes/settings/download-pipeline/clients'), 'DownloadClientsPage')
export const autoSearchRoute = lazyRoute('/settings/download-pipeline/auto-search', () => import('@/routes/settings/download-pipeline/auto-search'), 'AutoSearchPage')
export const rssSyncRoute = lazyRoute('/settings/download-pipeline/rss-sync', () => import('@/routes/settings/download-pipeline/rss-sync'), 'RssSyncPage')

// Settings — General (Server + Auth + Notifications)
export const generalSettingsRoute = redirectRoute('/settings/general', '/settings/general/server')
export const serverRoute = lazyRoute('/settings/general/server', () => import('@/routes/settings/general/server'), 'ServerPage')
export const authenticationRoute = lazyRoute('/settings/general/authentication', () => import('@/routes/settings/general/authentication'), 'AuthenticationPage')
export const notificationsRoute = lazyRoute('/settings/general/notifications', () => import('@/routes/settings/general/notifications'), 'NotificationsPage')

// Requests Admin
export const requestAdminRoute = redirectRoute('/requests-admin', '/requests-admin/queue')
export const requestQueueRoute = lazyRoute('/requests-admin/queue', () => import('@/routes/requests-admin/index'), 'RequestQueuePage')
export const requestUsersRoute = lazyRoute('/requests-admin/users', () => import('@/routes/requests-admin/users'), 'RequestUsersPage')
export const requestSettingsRoute = lazyRoute('/requests-admin/settings', () => import('@/routes/requests-admin/settings'), 'RequestSettingsPage')

// System
export const systemRoute = redirectRoute('/system', '/system/health')
export const healthRoute = lazyRoute('/system/health', () => import('@/routes/system/health'), 'SystemHealthPage')
export const tasksRoute = lazyRoute('/system/tasks', () => import('@/routes/system/tasks'), 'TasksPage')
export const logsRoute = lazyRoute('/system/logs', () => import('@/routes/system/logs'), 'LogsPage')
export const updateRoute = lazyRoute('/system/update', () => import('@/routes/system/update'), 'UpdatePage')

export const manualImportRoute = lazyRoute('/import', () => import('@/routes/import/index'), 'ManualImportPage')

// Dev
export const devControlsRoute = lazyRoute('/dev/controls', () => import('@/routes/dev/controls'), 'ControlsShowcasePage')

// Portal
export const portalLoginRoute = lazyRoute('/requests/auth/login', () => import('@/routes/requests/auth/login'), 'LoginPage')

export const portalSignupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/requests/auth/signup',
  component: lazyRouteComponent(() => import('@/routes/requests/auth/signup'), 'SignupPage'),
  validateSearch: (search: Record<string, unknown>): { token: string } => ({
    token: typeof search.token === 'string' ? search.token : '',
  }),
})

export const portalLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'portal-layout',
  component: () => (
    <PortalAuthGuard>
      <PortalLayout>
        <Outlet />
      </PortalLayout>
    </PortalAuthGuard>
  ),
})

const lazyPortalRoute = <P extends string>(
  path: P,
  importer: () => Promise<Record<string, unknown>>,
  exportName: string,
) =>
  createRoute({
    getParentRoute: () => portalLayoutRoute,
    path,
    component: lazyRouteComponent(importer, exportName),
  })

export const requestsListRoute = lazyPortalRoute('/requests', () => import('@/routes/requests/index'), 'RequestsListPage')
export const requestDetailRoute = lazyPortalRoute('/requests/$id', () => import('@/routes/requests/$id'), 'RequestDetailPage')
export const portalSettingsRoute = lazyPortalRoute('/requests/settings', () => import('@/routes/requests/settings'), 'PortalSettingsPage')
export const portalLibraryRoute = lazyPortalRoute('/requests/library', () => import('@/routes/requests/library'), 'PortalLibraryPage')

export const portalSearchRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests/search',
  component: lazyRouteComponent(() => import('@/routes/requests/search'), 'PortalSearchPageWrapper'),
  validateSearch: (search: Record<string, unknown>): { q: string } => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
})
