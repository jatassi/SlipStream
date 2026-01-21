import type { ReactNode } from 'react'
import { PortalHeader } from './PortalHeader'
import { PortalDownloads } from './PortalDownloads'

interface PortalLayoutProps {
  children: ReactNode
}

// Note: PortalLayout intentionally does NOT create its own QueryClient or call useWebSocketHandler().
// It inherits both from RootLayout which wraps all routes. This prevents:
// 1. Duplicate WebSocket message processing
// 2. Separate React Query caches between admin and portal views
export function PortalLayout({ children }: PortalLayoutProps) {
  return (
    <div className="flex h-dvh flex-col bg-background">
      <PortalHeader />
      <PortalDownloads />
      <main className="flex-1 overflow-auto">{children}</main>
    </div>
  )
}
