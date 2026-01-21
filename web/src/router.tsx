import {
  createRouter,
  createRoute,
  createRootRoute,
  Outlet,
} from '@tanstack/react-router'
import { RootLayout } from '@/components/layout/RootLayout'
import { PortalLayout, PortalAuthGuard } from '@/components/portal'

// Pages
import { DashboardPage } from '@/routes/index'
import { SearchPage } from '@/routes/search/index'
import { MoviesPage } from '@/routes/movies/index'
import { MovieDetailPage } from '@/routes/movies/$id'
import { AddMoviePage } from '@/routes/movies/add'
import { SeriesListPage } from '@/routes/series/index'
import { SeriesDetailPage } from '@/routes/series/$id'
import { AddSeriesPage } from '@/routes/series/add'
import { CalendarPage } from '@/routes/calendar/index'
import { MissingPage } from '@/routes/missing/index'
import { ActivityPage } from '@/routes/activity/index'
import { HistoryPage } from '@/routes/activity/history'
import { SettingsPage } from '@/routes/settings/index'
import { QualityProfilesPage } from '@/routes/settings/profiles'
import { RootFoldersPage } from '@/routes/settings/rootfolders'
import { IndexersPage } from '@/routes/settings/indexers'
import { DownloadClientsPage } from '@/routes/settings/downloadclients'
import { GeneralSettingsPage } from '@/routes/settings/general'
import { AutoSearchSettingsPage } from '@/routes/settings/autosearch'
import { ImportSettingsPage } from '@/routes/settings/import'
import { SlotsSettingsPage } from '@/routes/settings/slots'
import { NotificationsPage } from '@/routes/settings/notifications'
import { RequestQueuePage } from '@/routes/settings/requests/index'
import { RequestUsersPage } from '@/routes/settings/requests/users'
import { RequestSettingsPage } from '@/routes/settings/requests/settings'
import { ManualImportPage } from '@/routes/import/index'
import { TasksPage } from '@/routes/system/tasks'
import { SystemHealthPage } from '@/routes/system/health'

// Auth Pages
import { SetupPage } from '@/routes/auth/setup'

// Portal Pages
import { LoginPage } from '@/routes/requests/auth/login'
import { SignupPage } from '@/routes/requests/auth/signup'
import { RequestsListPage } from '@/routes/requests/index'
import { PortalSearchPageWrapper } from '@/routes/requests/search'
import { RequestDetailPage } from '@/routes/requests/$id'
import { PortalSettingsPage } from '@/routes/requests/settings'

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
    q: (search.q as string) || '',
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
  component: SettingsPage,
})

const profilesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/profiles',
  component: QualityProfilesPage,
})

const rootfoldersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/rootfolders',
  component: RootFoldersPage,
})

const indexersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/indexers',
  component: IndexersPage,
})

const downloadclientsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/downloadclients',
  component: DownloadClientsPage,
})

const generalSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/general',
  component: GeneralSettingsPage,
})

const autosearchSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/autosearch',
  component: AutoSearchSettingsPage,
})

const importSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/import',
  component: ImportSettingsPage,
})

const slotsSettingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/slots',
  component: SlotsSettingsPage,
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
    token: (search.token as string) || '',
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
    q: (search.q as string) || '',
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
  profilesRoute,
  rootfoldersRoute,
  indexersRoute,
  downloadclientsRoute,
  generalSettingsRoute,
  autosearchSettingsRoute,
  importSettingsRoute,
  slotsSettingsRoute,
  notificationsRoute,
  requestQueueRoute,
  requestUsersRoute,
  requestSettingsRoute,
  manualImportRoute,
  tasksRoute,
  healthRoute,
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
