import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { useNavigate, useLocation } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { usePortalAuthStore } from '@/stores'
import { useAuthStatus } from '@/hooks'

interface AdminAuthGuardProps {
  children: ReactNode
}

export function AdminAuthGuard({ children }: AdminAuthGuardProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { isAuthenticated, user, setRedirectUrl } = usePortalAuthStore()
  const { data: authStatus, isLoading: isLoadingStatus } = useAuthStatus()

  useEffect(() => {
    if (isLoadingStatus) return

    // First-time setup required
    if (authStatus?.requiresSetup) {
      navigate({ to: '/auth/setup' })
      return
    }

    // Not authenticated or not admin - redirect to login
    if (!isAuthenticated || !user?.isAdmin) {
      // Save the current URL for redirect after login
      if (location.pathname !== '/requests/auth/login' && location.pathname !== '/auth/setup') {
        setRedirectUrl(location.pathname + location.search)
      }
      navigate({ to: '/requests/auth/login' })
    }
  }, [isAuthenticated, user, authStatus, isLoadingStatus, navigate, location, setRedirectUrl])

  // Show loading while checking auth status
  if (isLoadingStatus) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Show loading during redirect
  if (authStatus?.requiresSetup || !isAuthenticated || !user?.isAdmin) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return <>{children}</>
}
