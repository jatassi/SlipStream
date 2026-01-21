import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { useNavigate, useLocation } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { usePortalAuthStore } from '@/stores'

interface PortalAuthGuardProps {
  children: ReactNode
}

export function PortalAuthGuard({ children }: PortalAuthGuardProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { isAuthenticated, setRedirectUrl } = usePortalAuthStore()

  useEffect(() => {
    if (!isAuthenticated) {
      if (location.pathname !== '/requests/auth/login' && location.pathname !== '/requests/auth/signup') {
        setRedirectUrl(location.pathname)
      }
      navigate({ to: '/requests/auth/login' })
    }
  }, [isAuthenticated, navigate, location.pathname, setRedirectUrl])

  if (!isAuthenticated) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return <>{children}</>
}
