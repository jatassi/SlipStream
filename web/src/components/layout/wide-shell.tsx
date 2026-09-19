import type { ReactNode } from 'react'

import { useRouterState } from '@tanstack/react-router'

import { Header } from './header'
import { Sidebar } from './sidebar'
import { paneFromPathname, useRecordLibraryModule } from './use-tab-nav'

export function WideShell({ children }: { children: ReactNode }) {
  useRecordLibraryModule()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const usesScreen = paneFromPathname(pathname) === 'dashboard' || pathname === '/more'

  return (
    <div className="flex h-dvh overflow-hidden">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Header />
        {usesScreen ? (
          <div className="relative min-h-0 flex-1 overflow-hidden">{children}</div>
        ) : (
          <main className="min-h-0 flex-1 overflow-auto p-6">{children}</main>
        )}
      </div>
    </div>
  )
}
