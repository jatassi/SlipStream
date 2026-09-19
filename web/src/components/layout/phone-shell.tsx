import type { ReactNode } from 'react'

import { useRouterState } from '@tanstack/react-router'

import { ErrorBoundary } from '@/components/error-boundary'
import { cn } from '@/lib/utils'

import { TabBar } from './tab-bar'
import { TabPanes } from './tab-panes'
import { paneFromPathname, tabFromPathname, useRecordLibraryModule } from './use-tab-nav'

const TAB_STACK_INSET = { paddingBottom: 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)' }

export function PhoneShell({ children }: { children: ReactNode }) {
  useRecordLibraryModule()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const pane = paneFromPathname(pathname)
  const currentTab = tabFromPathname(pathname)

  return (
    <div className="relative h-dvh overflow-hidden">
      <TabPanes pathname={pathname} />
      {pane === null ? <PhoneOverlay>{children}</PhoneOverlay> : null}
      <TabBar current={currentTab} />
    </div>
  )
}

function PhoneOverlay({ children }: { children: ReactNode }) {
  return (
    <div className={cn('absolute inset-0 z-10 scroll p-6')} style={TAB_STACK_INSET}>
      <ErrorBoundary>{children}</ErrorBoundary>
    </div>
  )
}
