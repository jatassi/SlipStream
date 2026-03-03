import { useEffect } from 'react'

import { useRouterState } from '@tanstack/react-router'

const ROUTE_TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/auth/setup': 'Setup',
  '/search': 'Search',
  '/movies': 'Movies',
  '/movies/add': 'Add Movie',
  '/series': 'Series',
  '/series/add': 'Add Series',
  '/calendar': 'Calendar',
  '/missing': 'Missing',
  '/downloads': 'Downloads',
  '/history': 'History',
  '/settings/media/root-folders': 'Settings - Media Management',
  '/settings/media/quality-profiles': 'Settings - Media Management',
  '/settings/media/version-slots': 'Settings - Media Management',
  '/settings/media/file-naming': 'Settings - Media Management',
  '/settings/media/arr-import': 'Settings - Media Management',
  '/settings/download-pipeline/indexers': 'Settings - Download Pipeline',
  '/settings/download-pipeline/clients': 'Settings - Download Pipeline',
  '/settings/download-pipeline/auto-search': 'Settings - Download Pipeline',
  '/settings/download-pipeline/rss-sync': 'Settings - Download Pipeline',
  '/settings/general/server': 'Settings - General',
  '/settings/general/authentication': 'Settings - General',
  '/settings/general/notifications': 'Settings - General',
  '/requests-admin': 'Requests',
  '/requests-admin/users': 'Requests - Users',
  '/requests-admin/settings': 'Requests - Settings',
  '/import': 'Manual Import',
  '/system/health': 'System - Health',
  '/system/logs': 'System - Logs',
  '/system/tasks': 'System - Tasks',
  '/system/update': 'System - Update',
  '/dev/controls': 'Dev - Controls',
  '/requests/auth/login': 'Login',
  '/requests/auth/signup': 'Sign Up',
  '/requests': 'Requests',
  '/requests/search': 'Requests - Search',
  '/requests/library': 'Requests - Library',
  '/requests/settings': 'Requests - Settings',
}

function getPageTitle(pathname: string): string {
  const exact = ROUTE_TITLES[pathname]
  if (exact) {return exact}

  if (pathname.startsWith('/movies/')) {return 'Movies'}
  if (pathname.startsWith('/series/')) {return 'Series'}
  if (pathname.startsWith('/requests/') && !pathname.startsWith('/requests/auth/'))
    {return 'Requests'}

  return ''
}

export function useDocumentTitle() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  useEffect(() => {
    const pageTitle = getPageTitle(pathname)
    document.title = pageTitle ? `SlipStream - ${pageTitle}` : 'SlipStream'
  }, [pathname])
}
