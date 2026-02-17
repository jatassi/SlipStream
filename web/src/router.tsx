import { createRouter } from '@tanstack/react-router'

import {
  activityRoute,
  addMovieRoute,
  addSeriesRoute,
  arrImportRoute,
  authenticationRoute,
  authSetupRoute,
  autoSearchRoute,
  calendarRoute,
  devColorsRoute,
  devControlsRoute,
  downloadClientsRoute,
  downloadsSettingsRoute,
  fileNamingRoute,
  healthRoute,
  historyRoute,
  indexersRoute,
  indexRoute,
  logsRoute,
  manualImportRoute,
  mediaSettingsRoute,
  missingRoute,
  movieDetailRoute,
  moviesRoute,
  notificationsRoute,
  portalLayoutRoute,
  portalLoginRoute,
  portalSearchRoute,
  portalSettingsRoute,
  portalSignupRoute,
  qualityProfilesRoute,
  requestDetailRoute,
  requestQueueRoute,
  requestSettingsRoute,
  requestsListRoute,
  requestUsersRoute,
  rootFoldersRoute,
  rootRoute,
  rssSyncRoute,
  searchRoute,
  seriesDetailRoute,
  seriesRoute,
  serverRoute,
  settingsRoute,
  systemSettingsRoute,
  tasksRoute,
  updateRoute,
  versionSlotsRoute,
} from '@/routes-config'

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
  arrImportRoute,
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
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions
  interface Register {
    router: typeof router
  }
}
