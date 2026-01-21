import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useLocation, useNavigate } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { Toaster } from '@/components/ui/sonner'
import { useWebSocketStore, useWebSocketHandler, useUIStore, useDevModeStore, usePortalAuthStore } from '@/stores'
import { useQueue, useStatus, useAuthStatus } from '@/hooks'

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 1,
    },
  },
})

interface RootLayoutProps {
  children: ReactNode
}

// Public paths that don't require admin auth
const PUBLIC_PATHS = [
  '/auth/setup',
  '/requests/auth/login',
  '/requests/auth/signup',
  '/requests',
  '/requests/',
]

function isPublicPath(pathname: string): boolean {
  // Check exact matches and prefix matches for portal routes
  return PUBLIC_PATHS.some(path =>
    pathname === path ||
    pathname.startsWith('/requests/') ||
    pathname.startsWith('/requests?')
  )
}

function LayoutContent({ children }: RootLayoutProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const { connect, disconnect } = useWebSocketStore()
  const { theme } = useUIStore()
  const { setEnabled: setDevModeEnabled, setSwitching: setDevModeSwitching } = useDevModeStore()
  const { isAuthenticated, user, setRedirectUrl } = usePortalAuthStore()
  const { data: status } = useStatus()
  const { data: authStatus, isLoading: isLoadingAuthStatus } = useAuthStatus()

  // Process WebSocket messages (handles progress events, query invalidation, etc.)
  useWebSocketHandler()

  // Keep downloading store synced globally (polls queue and syncs to store)
  // Only fetch for admin routes - portal users get queue updates via WebSocket
  const isPortalRoute = isPublicPath(location.pathname)
  useQueue(!isPortalRoute)

  // Sync developer mode state from backend on initial load and after switches
  // Also clears switching state in case WebSocket message was lost during reconnection
  useEffect(() => {
    if (status?.developerMode !== undefined) {
      setDevModeEnabled(status.developerMode)
      setDevModeSwitching(false)
    }
  }, [status?.developerMode, setDevModeEnabled, setDevModeSwitching])

  // Apply theme
  useEffect(() => {
    const root = document.documentElement
    if (theme === 'dark') {
      root.classList.add('dark')
    } else if (theme === 'light') {
      root.classList.remove('dark')
    } else {
      // System theme
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      if (mediaQuery.matches) {
        root.classList.add('dark')
      } else {
        root.classList.remove('dark')
      }
    }
  }, [theme])

  // Connect WebSocket on mount and handle page lifecycle events (for Safari mobile)
  useEffect(() => {
    connect()

    // Safari mobile can suspend WebSocket connections when page is hidden, leaving them
    // in a zombie state (readyState=OPEN but actually dead). Force reconnect when page
    // becomes visible to ensure we get real-time updates.
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        console.log('[WebSocket] Page visible, forcing reconnect')
        connect(true)
      }
    }

    // When Safari is killed via multitasking and reopened, the page is restored from
    // bfcache. The JS state is restored but WebSocket is dead. pageshow with persisted=true
    // indicates this scenario.
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

  // Handle unauthorized event from API client
  useEffect(() => {
    const handleUnauthorized = () => {
      const { logout } = usePortalAuthStore.getState()
      logout()
      navigate({ to: '/requests/auth/login' })
    }

    window.addEventListener('auth:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('auth:unauthorized', handleUnauthorized)
  }, [navigate])

  // Auth check for non-public routes
  useEffect(() => {
    // Skip auth check for public paths
    if (isPublicPath(location.pathname)) {
      return
    }

    // Wait for auth status to load
    if (isLoadingAuthStatus) {
      return
    }

    // If already authenticated as admin, skip setup/login redirects
    // This handles the case where authStatus is stale after setup
    if (isAuthenticated && user?.isAdmin) {
      return
    }

    // First-time setup required
    if (authStatus?.requiresSetup) {
      navigate({ to: '/auth/setup' })
      return
    }

    // Not authenticated or not admin - redirect to login
    if (!isAuthenticated || !user?.isAdmin) {
      setRedirectUrl(location.pathname + location.search)
      navigate({ to: '/requests/auth/login' })
    }
  }, [location.pathname, location.search, authStatus, isLoadingAuthStatus, isAuthenticated, user, navigate, setRedirectUrl])

  // Show loading while checking auth status for protected routes
  // But skip if already authenticated as admin (handles stale authStatus after setup)
  if (!isPublicPath(location.pathname) && isLoadingAuthStatus && !(isAuthenticated && user?.isAdmin)) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // For public paths, render without layout (setup, login, signup, portal)
  if (isPublicPath(location.pathname)) {
    return (
      <div className="min-h-screen bg-background">
        {children}
        <Toaster />
      </div>
    )
  }

  // Show loading while redirecting to auth
  // But if authenticated as admin, proceed (handles stale authStatus.requiresSetup)
  if (!isAuthenticated || !user?.isAdmin) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex h-screen bg-background">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
      <Toaster />
    </div>
  )
}

export function RootLayout({ children }: RootLayoutProps) {
  return (
    <QueryClientProvider client={queryClient}>
      <LayoutContent>{children}</LayoutContent>
    </QueryClientProvider>
  )
}
