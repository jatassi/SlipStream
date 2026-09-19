import { useEffect } from 'react'

import { useRouterState } from '@tanstack/react-router'
import type { LucideIcon } from 'lucide-react'
import { ArrowDownToLine, Ellipsis, LayoutDashboard, LayoutGrid, Search } from 'lucide-react'

import { getEnabledModules } from '@/modules'
import { useUIStore } from '@/stores'

export type TabId = 'dashboard' | 'library' | 'activity' | 'search' | 'more'
export type PaneId = 'dashboard' | 'movies' | 'series' | 'activity' | 'search' | 'more'

export const TAB_ITEMS: { id: TabId; label: string; icon: LucideIcon }[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'library', label: 'Library', icon: LayoutGrid },
  { id: 'activity', label: 'Activity', icon: ArrowDownToLine },
  { id: 'search', label: 'Search', icon: Search },
  { id: 'more', label: 'More', icon: Ellipsis },
]

export function libraryListPath(lastId: string | null): string {
  const modules = getEnabledModules()
  if (modules.length === 0) {
    throw new Error('No enabled modules')
  }
  const first = modules[0]
  const match = modules.find((mod) => mod.id === lastId)
  return (match ?? first).basePath
}

export function useLibraryListPath(): string {
  const lastId = useUIStore((s) => s.lastLibraryModuleId)
  return libraryListPath(lastId)
}

export function paneFromPathname(pathname: string): PaneId | null {
  if (pathname === '/') {
    return 'dashboard'
  }
  if (pathname === '/movies') {
    return 'movies'
  }
  if (pathname === '/series') {
    return 'series'
  }
  if (pathname === '/downloads') {
    return 'activity'
  }
  if (pathname === '/search') {
    return 'search'
  }
  if (pathname === '/more') {
    return 'more'
  }
  return null
}

function isLibraryPath(pathname: string): boolean {
  return getEnabledModules().some(
    (mod) => pathname === mod.basePath || pathname.startsWith(`${mod.basePath}/`),
  )
}

export function paneBehindPathname(pathname: string): PaneId {
  const pane = paneFromPathname(pathname)
  if (pane !== null) {
    return pane
  }
  return paneForCoveredTab(tabFromPathname(pathname), pathname)
}

function paneForCoveredTab(tab: TabId, pathname: string): PaneId {
  if (tab === 'dashboard') {
    return 'dashboard'
  }
  if (tab === 'library') {
    return pathname.startsWith('/series') ? 'series' : 'movies'
  }
  if (tab === 'activity') {
    return 'activity'
  }
  if (tab === 'search') {
    return 'search'
  }
  return 'more'
}

export function tabFromPathname(pathname: string): TabId {
  if (pathname === '/') {
    return 'dashboard'
  }
  if (isLibraryPath(pathname)) {
    return 'library'
  }
  if (pathname === '/downloads' || pathname.startsWith('/downloads/')) {
    return 'activity'
  }
  if (pathname === '/search' || pathname.startsWith('/search/')) {
    return 'search'
  }
  return 'more'
}

export function tabHref(id: TabId, libraryPath: string): string {
  if (id === 'dashboard') {
    return '/'
  }
  if (id === 'library') {
    return libraryPath
  }
  if (id === 'activity') {
    return '/downloads'
  }
  if (id === 'search') {
    return '/search'
  }
  return '/more'
}

export function useRecordLibraryModule() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const setLastLibraryModuleId = useUIStore((s) => s.setLastLibraryModuleId)

  useEffect(() => {
    const match = getEnabledModules().find(
      (mod) => pathname === mod.basePath || pathname.startsWith(`${mod.basePath}/`),
    )
    if (match) {
      setLastLibraryModuleId(match.id)
    }
  }, [pathname, setLastLibraryModuleId])
}
