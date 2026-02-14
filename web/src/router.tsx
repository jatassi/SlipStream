import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from '@tanstack/react-router'

import { RootLayout } from '@/components/layout/RootLayout'
import { PortalAuthGuard, PortalLayout } from '@/components/portal'
import { HistoryPage } from '@/routes/activity/history'
import { ActivityPage } from '@/routes/activity/index'
// Auth Pages
import { SetupPage } from '@/routes/auth/setup'
import { CalendarPage } from '@/routes/calendar/index'
// Dev Pages
import { ColorPreviewPage } from '@/routes/dev/colors'
import { ControlsShowcasePage } from '@/routes/dev/controls'
import { ManualImportPage } from '@/routes/import/index'
// Pages
import { DashboardPage } from '@/routes/index'
import { MissingPage } from '@/routes/missing/index'
import { MovieDetailPage } from '@/routes/movies/$id'
import { AddMoviePage } from '@/routes/movies/add'
import { MoviesPage } from '@/routes/movies/index'
import { RequestDetailPage } from '@/routes/requests/$id'
// Portal Pages
import { LoginPage } from '@/routes/requests/auth/login'
import { SignupPage } from '@/routes/requests/auth/signup'
import { RequestsListPage } from '@/routes/requests/index'
import { PortalSearchPageWrapper } from '@/routes/requests/search'
import { PortalSettingsPage } from '@/routes/requests/settings'
import { SearchPage } from '@/routes/search/index'
import { SeriesDetailPage } from '@/routes/series/$id'
import { AddSeriesPage } from '@/routes/series/add'
import { SeriesListPage } from '@/routes/series/index'
import { AutoSearchPage } from '@/routes/settings/downloads/auto-search'
import { DownloadClientsPage } from '@/routes/settings/downloads/clients'
// Downloads settings pages
import { IndexersPage } from '@/routes/settings/downloads/indexers'
import { RssSyncPage } from '@/routes/settings/downloads/rss-sync'
import { FileNamingPage } from '@/routes/settings/media/file-naming'
import { QualityProfilesPage } from '@/routes/settings/media/quality-profiles'
// Media settings pages
import { RootFoldersPage } from '@/routes/settings/media/root-folders'
import { VersionSlotsPage } from '@/routes/settings/media/version-slots'
// Other settings pages
import { NotificationsPage } from '@/routes/settings/notifications'
import { RequestQueuePage } from '@/routes/settings/requests/index'
import { RequestSettingsPage } from '@/routes/settings/requests/settings'
import { RequestUsersPage } from '@/routes/settings/requests/users'
import { AuthenticationPage } from '@/routes/settings/system/authentication'
// System settings pages
import { ServerPage } from '@/routes/settings/system/server'
import { SystemHealthPage } from '@/routes/system/health'
import { LogsPage } from '@/routes/system/logs'
import { TasksPage } from '@/routes/system/tasks'
import { UpdatePage } from '@/routes/system/update'

// Create root route with layout (auth is handled by RootLayout)
const rootRoute = createRootRoute({
  component: () => (
    <RootLayout>
      <Outlet />
    </RootLayout>
  ),
})

// Auth setup route (public, no auth required)
const authSetupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/setup',
  component: SetupPage,
})

// Dashboard
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardPage,
})

// Search route
const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/search',
  component: () => {
    const { q } = searchRoute.useSearch()
    return <SearchPage q={q} />
  },
  validateSearch: (search: Record<string, unknown>) => ({
    q: (search.q ?? '') as string,
  }),
})

// Movies routes
const moviesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/movies',
  component: MoviesPage,
})

const movieDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/movies/$id',
  component: MovieDetailPage,
})

const addMovieRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/movies/add',
  component: AddMoviePage,
})

// Series routes
const seriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series',
  component: SeriesListPage,
})

const seriesDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/$id',
  component: SeriesDetailPage,
})

const addSeriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/add',
  component: AddSeriesPage,
})

// Calendar route
const calendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/calendar',
  component: CalendarPage,
})

// Missing route
const missingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/missing',
  component: MissingPage,
})

// Activity routes
const activityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/activity',
  component: ActivityPage,
})

const historyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/activity/history',
  component: HistoryPage,
})

// Settings routes
const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  beforeLoad: () => {
    // eslint-disable-next-line @typescript-eslint/only-throw-error
    throw redirect({ to: '/settings/media/root-folders' })
  },
})

// Media settings routes
const mediaSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/media',
  beforeLoad: () => {
    // eslint-disable-next-line @typescript-eslint/only-throw-error
    throw redirect({ to: '/settings/media/root-folders' })
  },
})

const rootFoldersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/media/root-folders',
  component: RootFoldersPage,
})

const qualityProfilesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/media/quality-profiles',
  component: QualityProfilesPage,
})

const versionSlotsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/media/version-slots',
  component: VersionSlotsPage,
})

const fileNamingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/media/file-naming',
  component: FileNamingPage,
})

// Downloads settings routes
const downloadsSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloads',
  beforeLoad: () => {
    // eslint-disable-next-line @typescript-eslint/only-throw-error
    throw redirect({ to: '/settings/downloads/indexers' })
  },
})

const indexersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloads/indexers',
  component: IndexersPage,
})

const downloadClientsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloads/clients',
  component: DownloadClientsPage,
})

const autoSearchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloads/auto-search',
  component: AutoSearchPage,
})

const rssSyncRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloads/rss-sync',
  component: RssSyncPage,
})

// System settings routes
const systemSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/system',
  beforeLoad: () => {
    // eslint-disable-next-line @typescript-eslint/only-throw-error
    throw redirect({ to: '/settings/system/server' })
  },
})

const serverRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/system/server',
  component: ServerPage,
})

const authenticationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/system/authentication',
  component: AuthenticationPage,
})

const updateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system/update',
  component: UpdatePage,
})

const notificationsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/notifications',
  component: NotificationsPage,
})

// Admin request settings routes
const requestQueueRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/requests',
  component: RequestQueuePage,
})

const requestUsersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/requests/users',
  component: RequestUsersPage,
})

const requestSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/requests/settings',
  component: RequestSettingsPage,
})

// Import routes
const manualImportRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/import',
  component: ManualImportPage,
})

// System routes
const tasksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system/tasks',
  component: TasksPage,
})

const healthRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system/health',
  component: SystemHealthPage,
})

const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system/logs',
  component: LogsPage,
})

// Dev routes
const devColorsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dev/colors',
  component: ColorPreviewPage,
})

const devControlsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dev/controls',
  component: ControlsShowcasePage,
})

// Portal auth routes (no layout)
const portalLoginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/requests/auth/login',
  component: LoginPage,
})

const portalSignupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/requests/auth/signup',
  component: SignupPage,
  validateSearch: (search: Record<string, unknown>) => ({
    token: (search.token ?? '') as string,
  }),
})

// Portal layout route (protected)
const portalLayoutRoute = createRoute({
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

// Portal routes (protected, with layout)
const requestsListRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests',
  component: RequestsListPage,
})

const portalSearchRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests/search',
  component: PortalSearchPageWrapper,
  validateSearch: (search: Record<string, unknown>) => ({
    q: (search.q ?? '') as string,
  }),
})

const requestDetailRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests/$id',
  component: RequestDetailPage,
})

const portalSettingsRoute = createRoute({
  getParentRoute: () => portalLayoutRoute,
  path: '/requests/settings',
  component: PortalSettingsPage,
})

// Build route tree
const routeTree = rootRoute.addChildren([
  // Public auth routes
  authSetupRoute,
  // Main app routes (auth handled by RootLayout)
  indexRoute,
  searchRoute,
  moviesRoute,
  movieDetailRoute,
  addMovieRoute,
  seriesRoute,
  seriesDetailRoute,
  addSeriesRoute,
  calendarRoute,
  missingRoute,
  activityRoute,
  historyRoute,
  settingsRoute,
  // Media settings
  mediaSettingsRoute,
  rootFoldersRoute,
  qualityProfilesRoute,
  versionSlotsRoute,
  fileNamingRoute,
  // Downloads settings
  downloadsSettingsRoute,
  indexersRoute,
  downloadClientsRoute,
  autoSearchRoute,
  rssSyncRoute,
  // System settings
  systemSettingsRoute,
  serverRoute,
  authenticationRoute,
  // Other settings
  notificationsRoute,
  requestQueueRoute,
  requestUsersRoute,
  requestSettingsRoute,
  manualImportRoute,
  tasksRoute,
  healthRoute,
  logsRoute,
  updateRoute,
  // Dev routes
  devColorsRoute,
  devControlsRoute,
  // Portal auth routes (public)
  portalLoginRoute,
  portalSignupRoute,
  // Portal routes (with PortalAuthGuard)
  portalLayoutRoute.addChildren([
    requestsListRoute,
    portalSearchRoute,
    requestDetailRoute,
    portalSettingsRoute,
  ]),
])

// Create router
export const router = createRouter({ routeTree })

// Type declaration for router
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
