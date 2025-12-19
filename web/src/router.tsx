import {
  createRouter,
  createRoute,
  createRootRoute,
  Outlet,
} from '@tanstack/react-router'
import { RootLayout } from '@/components/layout/RootLayout'

// Pages
import { DashboardPage } from '@/routes/index'
import { MoviesPage } from '@/routes/movies/index'
import { MovieDetailPage } from '@/routes/movies/$id'
import { AddMoviePage } from '@/routes/movies/add'
import { SeriesListPage } from '@/routes/series/index'
import { SeriesDetailPage } from '@/routes/series/$id'
import { AddSeriesPage } from '@/routes/series/add'
import { ActivityPage } from '@/routes/activity/index'
import { HistoryPage } from '@/routes/activity/history'
import { SettingsPage } from '@/routes/settings/index'
import { QualityProfilesPage } from '@/routes/settings/profiles'
import { RootFoldersPage } from '@/routes/settings/rootfolders'
import { IndexersPage } from '@/routes/settings/indexers'
import { DownloadClientsPage } from '@/routes/settings/downloadclients'
import { GeneralSettingsPage } from '@/routes/settings/general'

// Create root route with layout
const rootRoute = createRootRoute({
  component: () => (
    <RootLayout>
      <Outlet />
    </RootLayout>
  ),
})

// Dashboard
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardPage,
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

// Build route tree
const routeTree = rootRoute.addChildren([
  indexRoute,
  moviesRoute,
  movieDetailRoute,
  addMovieRoute,
  seriesRoute,
  seriesDetailRoute,
  addSeriesRoute,
  activityRoute,
  historyRoute,
  settingsRoute,
  profilesRoute,
  rootfoldersRoute,
  indexersRoute,
  downloadclientsRoute,
  generalSettingsRoute,
])

// Create router
export const router = createRouter({ routeTree })

// Type declaration for router
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
