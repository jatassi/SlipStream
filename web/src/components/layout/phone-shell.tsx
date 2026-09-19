import type { ReactNode } from 'react'

import { useRouterState } from '@tanstack/react-router'

import { PhoneOverlay } from './phone-overlay'
import { isScreenFillPath } from './push-routes'
import { TabBar } from './tab-bar'
import { TabPanes } from './tab-panes'
import { usePushBack } from './use-push-back'
import { usePushLayer } from './use-push-layer'
import { usePushMotion } from './use-push-motion'
import { paneFromPathname, tabFromPathname, useRecordLibraryModule } from './use-tab-nav'

import './phone-shell.css'

export function PhoneShell({ children }: { children: ReactNode }) {
  useRecordLibraryModule()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const pane = paneFromPathname(pathname)
  const overlay = pane === null
  const layer = usePushLayer({
    active: overlay,
    pathname,
    animate: usePushMotion(pathname),
  })
  const back = usePushBack()
  const currentTab = tabFromPathname(pathname)

  return (
    <div className="relative h-dvh overflow-hidden">
      <div
        className="phone-root absolute inset-0"
        data-behind={overlay ? '' : undefined}
        data-instant={layer.instant ? '' : undefined}
      >
        <TabPanes pathname={pathname} overlay={layer.mounted} />
      </div>
      {layer.mounted ? (
        <PhoneOverlay
          fill={isScreenFillPath(layer.path)}
          back={overlayBack(layer.path, back)}
          exiting={layer.exiting}
          instant={layer.instant}
          onExitEnd={layer.finishExit}
        >
          {children}
        </PhoneOverlay>
      ) : null}
      <TabBar current={currentTab} />
    </div>
  )
}

function overlayBack(
  path: string,
  back: { label: string; onClick: () => void } | undefined,
): { label: string; onClick: () => void } | undefined {
  if (isScreenFillPath(path) || back === undefined) {
    return undefined
  }
  return back
}
