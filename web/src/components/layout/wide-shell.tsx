import type { ReactNode } from 'react'

import { useRouterState } from '@tanstack/react-router'

import { BackControl } from '@/components/screen/screen'

import { Header } from './header'
import { isScreenFillPath } from './push-routes'
import { Sidebar } from './sidebar'
import { usePushBack } from './use-push-back'
import { useRecordLibraryModule } from './use-tab-nav'

export function WideShell({ children }: { children: ReactNode }) {
  useRecordLibraryModule()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const fill = isScreenFillPath(pathname)
  const pushBack = usePushBack()
  const back = chromeBack(pathname, pushBack)

  return (
    <div className="flex h-dvh overflow-hidden">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <Header />
        {fill ? (
          <div className="relative min-h-0 flex-1 overflow-hidden">{children}</div>
        ) : (
          <main className="min-h-0 flex-1 overflow-auto p-6">
            {back === undefined ? null : (
              <BackControl label={back.label} onClick={back.onClick} className="-ml-3 mb-2" />
            )}
            {children}
          </main>
        )}
      </div>
    </div>
  )
}

function chromeBack(
  pathname: string,
  back: { label: string; onClick: () => void } | undefined,
): { label: string; onClick: () => void } | undefined {
  if (isScreenFillPath(pathname) || back === undefined) {
    return undefined
  }
  return back
}
