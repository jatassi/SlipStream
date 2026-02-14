import type { ReactNode } from 'react'
import { useEffect } from 'react'

import { useLocation, useNavigate } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'

import { useAuthStatus } from '@/hooks'
import { usePortalAuthStore } from '@/stores'

type AdminAuthGuardProps = {
  children: ReactNode
}

export function AdminAuthGuard({ children }: AdminAuthGuardProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { isAuthenticated, user, setRedirectUrl } = usePortalAuthStore()
  const { data: authStatus, isLoading: isLoadingStatus } = useAuthStatus()

  useEffect(() => {
    if (isLoadingStatus) {
      return
    }

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
      <div className="bg-background flex min-h-screen items-center justify-center">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  // Show loading during redirect
  if (authStatus?.requiresSetup || !isAuthenticated || !user?.isAdmin) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  return children
}
