import { paneFromPathname, type PaneId } from './use-tab-nav'

type SettingsSection = {
  path: string
  title: string
}

const SETTINGS_SECTIONS: SettingsSection[] = [
  { path: '/settings/media', title: 'Media' },
  { path: '/settings/download-pipeline', title: 'Download Pipeline' },
  { path: '/settings/general', title: 'General' },
]

function isAddPath(pathname: string): boolean {
  return pathname === '/movies/add' || pathname === '/series/add'
}

function isMovieDetailPath(pathname: string): boolean {
  return pathname.startsWith('/movies/') && pathname !== '/movies/add'
}

function isSeriesDetailPath(pathname: string): boolean {
  return pathname.startsWith('/series/') && pathname !== '/series/add'
}

function isDetailPath(pathname: string): boolean {
  return isMovieDetailPath(pathname) || isSeriesDetailPath(pathname)
}

function isLibraryPushPath(pathname: string): boolean {
  return pathname.startsWith('/movies/') || pathname.startsWith('/series/')
}

function isSettingsPath(pathname: string): boolean {
  return pathname === '/settings' || pathname.startsWith('/settings/')
}

const SYSTEM_INDEX_PATH = '/system/health'

function isRequestsAdminPath(pathname: string): boolean {
  return pathname.startsWith('/requests-admin')
}

function isSystemPath(pathname: string): boolean {
  return pathname === '/system' || pathname.startsWith('/system/')
}

function isSystemSubScreen(pathname: string): boolean {
  return isSystemPath(pathname) && pathname !== SYSTEM_INDEX_PATH
}

function settingsSectionForLeaf(pathname: string): SettingsSection | undefined {
  return SETTINGS_SECTIONS.find((section) => pathname.startsWith(`${section.path}/`))
}

export function backLabelForPathname(pathname: string): string | undefined {
  if (paneFromPathname(pathname) !== null) {
    return undefined
  }
  if (isLibraryPushPath(pathname)) {
    return 'Library'
  }
  if (isSystemSubScreen(pathname)) {
    return 'System'
  }
  return settingsSectionForLeaf(pathname)?.title ?? 'More'
}

const SCREEN_FILL_PANES = new Set<PaneId>(['dashboard', 'activity', 'movies', 'series', 'search', 'more'])
const SCREEN_FILL_PATHS = new Set(['/calendar', '/import', '/missing', '/history'])

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (SCREEN_FILL_PATHS.has(pathname) || (pane !== null && SCREEN_FILL_PANES.has(pane))) {
    return true
  }
  return (
    isAddPath(pathname) ||
    isRequestsAdminPath(pathname) ||
    isDetailPath(pathname) ||
    isSettingsPath(pathname) ||
    isSystemPath(pathname)
  )
}
