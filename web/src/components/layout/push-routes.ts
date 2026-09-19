import { paneFromPathname } from './use-tab-nav'

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

export function backLabelForPathname(pathname: string): string | undefined {
  if (paneFromPathname(pathname) !== null) {
    return undefined
  }
  if (isLibraryPushPath(pathname)) {
    return 'Library'
  }
  return 'More'
}

export function isScreenFillPath(pathname: string): boolean {
  const pane = paneFromPathname(pathname)
  if (pathname === '/more' || pane === 'dashboard' || pane === 'activity') {
    return true
  }
  return isDetailPath(pathname)
}
