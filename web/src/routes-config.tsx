import type React from 'react'

import { createRootRoute, createRoute, Outlet, redirect } from '@tanstack/react-router'

import { RootLayout } from '@/components/layout/root-layout'
import { PortalAuthGuard, PortalLayout } from '@/components/portal'
import { SetupPage } from '@/routes/auth/setup'
import { CalendarPage } from '@/routes/calendar/index'
import { ColorPreviewPage } from '@/routes/dev/colors'
import { ControlsShowcasePage } from '@/routes/dev/controls'
import { ActivityPage } from '@/routes/downloads/index'
import { HistoryPage } from '@/routes/history/history'
import { ManualImportPage } from '@/routes/import/index'
import { DashboardPage } from '@/routes/index'
import { MissingPage } from '@/routes/missing/index'
import { MovieDetailPage } from '@/routes/movies/$id'
import { AddMoviePage } from '@/routes/movies/add'
import { MoviesPage } from '@/routes/movies/index'
import { RequestDetailPage } from '@/routes/requests/$id'
import { LoginPage } from '@/routes/requests/auth/login'
import { SignupPage } from '@/routes/requests/auth/signup'
import { RequestsListPage } from '@/routes/requests/index'
import { PortalLibraryPage } from '@/routes/requests/library'
import { PortalSearchPageWrapper } from '@/routes/requests/search'
import { PortalSettingsPage } from '@/routes/requests/settings'
import { RequestQueuePage } from '@/routes/requests-admin/index'
import { RequestSettingsPage } from '@/routes/requests-admin/settings'
import { RequestUsersPage } from '@/routes/requests-admin/users'
import { SearchPage } from '@/routes/search/index'
import { SeriesDetailPage } from '@/routes/series/$id'
import { AddSeriesPage } from '@/routes/series/add'
import { SeriesListPage } from '@/routes/series/index'
import { AutoSearchPage } from '@/routes/settings/download-pipeline/auto-search'
import { DownloadClientsPage } from '@/routes/settings/download-pipeline/clients'
import { IndexersPage } from '@/routes/settings/download-pipeline/indexers'
import { RssSyncPage } from '@/routes/settings/download-pipeline/rss-sync'
import { AuthenticationPage } from '@/routes/settings/general/authentication'
import { NotificationsPage } from '@/routes/settings/general/notifications'
import { ServerPage } from '@/routes/settings/general/server'
import { ArrImportPage } from '@/routes/settings/media/arr-import'
import { FileNamingPage } from '@/routes/settings/media/file-naming'
import { QualityProfilesPage } from '@/routes/settings/media/quality-profiles'
import { RootFoldersPage } from '@/routes/settings/media/root-folders'
import { VersionSlotsPage } from '@/routes/settings/media/version-slots'
import { SystemHealthPage } from '@/routes/system/health'
import { LogsPage } from '@/routes/system/logs'
import { TasksPage } from '@/routes/system/tasks'
import { UpdatePage } from '@/routes/system/update'

export const rootRoute = createRootRoute({
  component: () => (
    <RootLayout>
      <Outlet />
    </RootLayout>
  ),
})

const route = <P extends string>(path: P, component: () => React.JSX.Element) =>
  createRoute({ getParentRoute: () => rootRoute, path, component })

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

export const authSetupRoute = route('/auth/setup', SetupPage)
export const indexRoute = route('/', DashboardPage)

export const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/search',
  component: () => {
    const search: { q: string } = searchRoute.useSearch()
    return <SearchPage q={search.q} />
  },
  validateSearch: (search: Record<string, unknown>): { q: string } => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
})

export const moviesRoute = route('/movies', MoviesPage)
export const movieDetailRoute = route('/movies/$id', MovieDetailPage)
export const addMovieRoute = route('/movies/add', AddMoviePage)
export const seriesRoute = route('/series', SeriesListPage)
export const seriesDetailRoute = route('/series/$id', SeriesDetailPage)
export const addSeriesRoute = route('/series/add', AddSeriesPage)
export const calendarRoute = route('/calendar', CalendarPage)
export const missingRoute = route('/missing', MissingPage)
export const activityRoute = route('/downloads', ActivityPage)
export const historyRoute = route('/history', HistoryPage)


// Settings — Media
export const settingsRoute = redirectRoute('/settings', '/settings/media/root-folders')
export const mediaSettingsRoute = redirectRoute('/settings/media', '/settings/media/root-folders')
export const rootFoldersRoute = route('/settings/media/root-folders', RootFoldersPage)
export const qualityProfilesRoute = route('/settings/media/quality-profiles', QualityProfilesPage)
export const versionSlotsRoute = route('/settings/media/version-slots', VersionSlotsPage)
export const fileNamingRoute = route('/settings/media/file-naming', FileNamingPage)
export const arrImportRoute = route('/settings/media/arr-import', ArrImportPage)

// Settings — Download Pipeline
export const downloadPipelineRoute = redirectRoute('/settings/download-pipeline', '/settings/download-pipeline/indexers')
export const indexersRoute = route('/settings/download-pipeline/indexers', IndexersPage)
export const downloadClientsRoute = route('/settings/download-pipeline/clients', DownloadClientsPage)
export const autoSearchRoute = route('/settings/download-pipeline/auto-search', AutoSearchPage)
export const rssSyncRoute = route('/settings/download-pipeline/rss-sync', RssSyncPage)

// Settings — General (Server + Auth + Notifications)
export const generalSettingsRoute = redirectRoute('/settings/general', '/settings/general/server')
export const serverRoute = route('/settings/general/server', ServerPage)
export const authenticationRoute = route('/settings/general/authentication', AuthenticationPage)
export const notificationsRoute = route('/settings/general/notifications', NotificationsPage)

// Requests Admin
export const requestQueueRoute = route('/requests-admin', RequestQueuePage)
export const requestUsersRoute = route('/requests-admin/users', RequestUsersPage)
export const requestSettingsRoute = route('/requests-admin/settings', RequestSettingsPage)

// System
export const systemRoute = redirectRoute('/system', '/system/health')
export const healthRoute = route('/system/health', SystemHealthPage)
export const tasksRoute = route('/system/tasks', TasksPage)
export const logsRoute = route('/system/logs', LogsPage)
export const updateRoute = route('/system/update', UpdatePage)

export const manualImportRoute = route('/import', ManualImportPage)


// Dev
export const devColorsRoute = route('/dev/colors', ColorPreviewPage)
export const devControlsRoute = route('/dev/controls', ControlsShowcasePage)

// Portal
export const portalLoginRoute = route('/requests/auth/login', LoginPage)

export const portalSignupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/requests/auth/signup',
  component: SignupPage,
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

const portalRoute = <P extends string>(path: P, component: () => React.JSX.Element) =>
  createRoute({ getParentRoute: () => portalLayoutRoute, path, component })

export const requestsListRoute = portalRoute('/requests', RequestsListPage)
export const requestDetailRoute = portalRoute('/requests/$id', RequestDetailPage)
export const portalSettingsRoute = portalRoute('/requests/settings', PortalSettingsPage)
export const portalLibraryRoute = portalRoute('/requests/library', PortalLibraryPage)

export const portalSearchRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests/search',
  component: PortalSearchPageWrapper,
  validateSearch: (search: Record<string, unknown>): { q: string } => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
})
