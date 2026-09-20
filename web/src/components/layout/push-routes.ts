<<<<<<< HEAD
import { paneFromPathname, type PaneId } from './use-tab-nav'
=======
import type { PaneId } from './use-tab-nav'
import { paneFromPathname } from './use-tab-nav'
>>>>>>> worktree-agent-a67f3206ac9fd8c76

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

export function isRequestsAdminPath(pathname: string): boolean {
  return pathname.startsWith('/requests-admin')
}

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

<<<<<<< HEAD
<<<<<<< HEAD
const FILL_PANES = new Set(['dashboard', 'activity', 'movies', 'series'])
=======
const FILL_PANES = new Set<PaneId>(['dashboard', 'activity', 'movies', 'series'])
>>>>>>> worktree-agent-a67f3206ac9fd8c76
const FILL_PATHS = new Set(['/more', '/calendar', '/import'])

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (pane !== null && FILL_PANES.has(pane)) {
    return true
  }
  if (FILL_PATHS.has(pathname)) {
<<<<<<< HEAD
=======
const SCREEN_FILL_PANES = new Set<PaneId>(['dashboard', 'activity', 'movies', 'series', 'search', 'more'])

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (pane !== null && SCREEN_FILL_PANES.has(pane)) {
>>>>>>> worktree-agent-a1500afee18095a29
    return true
  }
  return (
    isAddPath(pathname) ||
=======
    return true
  }
  return (
    isRequestsAdminPath(pathname) ||
>>>>>>> worktree-agent-a67f3206ac9fd8c76
    isDetailPath(pathname) ||
    isSettingsPath(pathname) ||
    isSystemPath(pathname)
  )
}
