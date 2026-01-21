import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { useNavigate, useLocation } from '@tanstack/react-router'
import { Loader2, Ban } from 'lucide-react'
import { usePortalAuthStore } from '@/stores'
import { usePortalEnabled } from '@/hooks'

interface PortalAuthGuardProps {
  children: ReactNode
}

export function PortalAuthGuard({ children }: PortalAuthGuardProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { isAuthenticated, setRedirectUrl } = usePortalAuthStore()
  const portalEnabled = usePortalEnabled()

  useEffect(() => {
    if (!isAuthenticated) {
      if (location.pathname !== '/requests/auth/login' && location.pathname !== '/requests/auth/signup') {
        setRedirectUrl(location.pathname)
      }
      navigate({ to: '/requests/auth/login' })
    }
  }, [isAuthenticated, navigate, location.pathname, setRedirectUrl])

  if (!portalEnabled) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="text-center space-y-4">
          <Ban className="size-12 mx-auto text-muted-foreground" />
          <h1 className="text-xl font-semibold">Requests Portal Disabled</h1>
          <p className="text-muted-foreground max-w-sm">
            The external requests portal is currently disabled. Please contact your server administrator.
          </p>
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return <>{children}</>
}
