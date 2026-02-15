import type { ReactNode } from 'react'
import { useEffect } from 'react'

import { useLocation, useNavigate } from '@tanstack/react-router'
import { Ban, Loader2 } from 'lucide-react'

import { usePortalEnabled } from '@/hooks'
import { usePortalAuthStore } from '@/stores'

type PortalAuthGuardProps = {
  children: ReactNode
}

export function PortalAuthGuard({ children }: PortalAuthGuardProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { isAuthenticated, setRedirectUrl } = usePortalAuthStore()
  const portalEnabled = usePortalEnabled()

  useEffect(() => {
    if (!isAuthenticated) {
      if (
        location.pathname !== '/requests/auth/login' &&
        location.pathname !== '/requests/auth/signup'
      ) {
        setRedirectUrl(location.pathname)
      }
      void navigate({ to: '/requests/auth/login' })
    }
  }, [isAuthenticated, navigate, location.pathname, setRedirectUrl])

  if (!portalEnabled) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <div className="space-y-4 text-center">
          <Ban className="text-muted-foreground mx-auto size-12" />
          <h1 className="text-xl font-semibold">Requests Portal Disabled</h1>
          <p className="text-muted-foreground max-w-sm">
            The external requests portal is currently disabled. Please contact your server
            administrator.
          </p>
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  return children
}
