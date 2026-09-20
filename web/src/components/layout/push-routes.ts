import { paneFromPathname } from './use-tab-nav'

export type SettingsSection = {
  path: string
  title: string
}

export const SETTINGS_SECTIONS: SettingsSection[] = [
  { path: '/settings/media', title: 'Media' },
  { path: '/settings/download-pipeline', title: 'Download Pipeline' },
  { path: '/settings/general', title: 'General' },
]

export function isAddPath(pathname: string): boolean {
  return pathname === '/movies/add' || pathname === '/series/add'
}

export function isMovieDetailPath(pathname: string): boolean {
  return pathname.startsWith('/movies/') && pathname !== '/movies/add'
}

export function isSeriesDetailPath(pathname: string): boolean {
  return pathname.startsWith('/series/') && pathname !== '/series/add'
}

export function isDetailPath(pathname: string): boolean {
  return isMovieDetailPath(pathname) || isSeriesDetailPath(pathname)
}

export function isLibraryPushPath(pathname: string): boolean {
  return pathname.startsWith('/movies/') || pathname.startsWith('/series/')
}

export function isSettingsPath(pathname: string): boolean {
  return pathname === '/settings' || pathname.startsWith('/settings/')
}

export const SYSTEM_INDEX_PATH = '/system/health'

export function isSystemPath(pathname: string): boolean {
  return pathname === '/system' || pathname.startsWith('/system/')
}

function isSystemSubScreen(pathname: string): boolean {
  return isSystemPath(pathname) && pathname !== SYSTEM_INDEX_PATH
}

export function settingsSectionForLeaf(pathname: string): SettingsSection | undefined {
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

const FILL_PANES = new Set(['dashboard', 'activity', 'movies', 'series'])
const FILL_PATHS = new Set(['/more', '/calendar', '/import'])

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (pane !== null && FILL_PANES.has(pane)) {
    return true
  }
  if (FILL_PATHS.has(pathname)) {
    return true
  }
  return (
    isAddPath(pathname) ||
    isDetailPath(pathname) ||
    isSettingsPath(pathname) ||
    isSystemPath(pathname)
  )
}
