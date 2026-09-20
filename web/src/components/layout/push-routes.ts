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
  return settingsSectionForLeaf(pathname)?.title ?? 'More'
}

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (pathname === '/more' || pane === 'dashboard' || pane === 'activity') {
    return true
  }
  if (pathname === '/calendar' || pathname === '/import') {
    return true
  }
  return isDetailPath(pathname) || isSettingsPath(pathname)
}
