import { useEffect } from 'react'

import { useLocation, useNavigate } from '@tanstack/react-router'

import { useAuthStatus, useQueue, useStatus } from '@/hooks'
import {
  useDevModeStore,
  usePortalAuthStore,
  useUIStore,
  useWebSocketHandler,
  useWebSocketStore,
} from '@/stores'

const PUBLIC_PATHS = [
  '/auth/setup',
  '/requests/auth/login',
  '/requests/auth/signup',
  '/requests',
  '/requests/',
]

export function isPublicPath(pathname: string): boolean {
  return PUBLIC_PATHS.some(
    (path) =>
      pathname === path || pathname.startsWith('/requests/') || pathname.startsWith('/requests?'),
  )
}

function useWebSocketLifecycle() {
  const { connect, disconnect } = useWebSocketStore()

  useEffect(() => {
    connect()

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        console.log('[WebSocket] Page visible, forcing reconnect')
        connect(true)
      }
    }

    const handlePageShow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        console.log('[WebSocket] Page restored from bfcache, forcing reconnect')
        connect(true)
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    window.addEventListener('pageshow', handlePageShow)

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      window.removeEventListener('pageshow', handlePageShow)
      disconnect()
    }
  }, [connect, disconnect])
}

function useThemeEffect() {
  const { theme } = useUIStore()

  useEffect(() => {
    const root = document.documentElement
    if (theme === 'dark') {
      root.classList.add('dark')
    } else if (theme === 'light') {
      root.classList.remove('dark')
    } else {
      const mediaQuery = globalThis.matchMedia('(prefers-color-scheme: dark)')
      root.classList.toggle('dark', mediaQuery.matches)
    }
  }, [theme])
}

function useDevModeSync() {
  const { setEnabled, setSwitching } = useDevModeStore()
  const { data: status } = useStatus()

  useEffect(() => {
    if (status?.developerMode !== undefined) {
      setEnabled(status.developerMode)
      setSwitching(false)
    }
  }, [status?.developerMode, setEnabled, setSwitching])
}

function useUnauthorizedHandler() {
  const navigate = useNavigate()

  useEffect(() => {
    const handleUnauthorized = () => {
      const { logout } = usePortalAuthStore.getState()
      logout()
      void navigate({ to: '/requests/auth/login' })
    }

    globalThis.addEventListener('auth:unauthorized', handleUnauthorized)
    return () => globalThis.removeEventListener('auth:unauthorized', handleUnauthorized)
  }, [navigate])
}

function getAuthRedirect(opts: {
  pathname: string
  isLoadingAuthStatus: boolean
  isAuthenticated: boolean
  isAdmin: boolean
  requiresSetup: boolean
}): string | null {
  if (isPublicPath(opts.pathname)) {
    return null
  }
  if (opts.isLoadingAuthStatus) {
    return null
  }
  if (opts.isAuthenticated && opts.isAdmin) {
    return null
  }
  if (opts.requiresSetup) {
    return '/auth/setup'
  }
  if (!opts.isAuthenticated || !opts.isAdmin) {
    return '/requests/auth/login'
  }
  return null
}

function useAuthRedirect() {
  const location = useLocation()
  const navigate = useNavigate()
  const { isAuthenticated, user, setRedirectUrl } = usePortalAuthStore()
  const { data: authStatus, isLoading: isLoadingAuthStatus } = useAuthStatus()

  useEffect(() => {
    const redirect = getAuthRedirect({
      pathname: location.pathname,
      isLoadingAuthStatus,
      isAuthenticated,
      isAdmin: user?.isAdmin ?? false,
      requiresSetup: authStatus?.requiresSetup ?? false,
    })

    if (redirect === null) {
      return
    }

    if (redirect === '/requests/auth/login') {
      setRedirectUrl(`${location.pathname}${location.searchStr}`)
    }

    void navigate({ to: redirect })
  }, [
    location.pathname,
    location.search,
    location.searchStr,
    authStatus,
    isLoadingAuthStatus,
    isAuthenticated,
    user,
    navigate,
    setRedirectUrl,
  ])

  return { isAuthenticated, user, isLoadingAuthStatus }
}

export function useLayoutEffects() {
  const location = useLocation()

  useWebSocketHandler()

  const isPortalRoute = isPublicPath(location.pathname)
  useQueue(!isPortalRoute)

  useWebSocketLifecycle()
  useThemeEffect()
  useDevModeSync()
  useUnauthorizedHandler()
  const auth = useAuthRedirect()

  return {
    pathname: location.pathname,
    isPublicRoute: isPortalRoute,
    isAuthenticated: auth.isAuthenticated,
    isAdmin: auth.user?.isAdmin ?? false,
    isLoadingAuth: auth.isLoadingAuthStatus,
  }
}
