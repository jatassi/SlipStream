/* eslint-disable react-hooks/refs */
import { type ReactNode, useCallback, useRef, useState } from 'react'

type PushPhase = 'off' | 'on' | 'out'

type PushLayer = {
  mounted: boolean
  exiting: boolean
  instant: boolean
  path: string
  node: ReactNode
  finishExit: () => void
}

type LayerSetters = {
  setPhase: (phase: PushPhase) => void
  setShownPath: (path: string) => void
  setInstant: (instant: boolean) => void
}

export function usePushLayer({
  active,
  pathname,
  children,
  animate,
}: {
  active: boolean
  pathname: string
  children: ReactNode
  animate: boolean
}): PushLayer {
  const snap = useRef(children)
  const [phase, setPhase] = useState<PushPhase>(active ? 'on' : 'off')
  const [shownPath, setShownPath] = useState(pathname)
  const [instant, setInstant] = useState(!animate)
  const setters: LayerSetters = { setPhase, setShownPath, setInstant }

  if (active) {
    snap.current = children
    syncEnter({ phase, shownPath, pathname, animate, setters })
  } else {
    syncExit({ phase, animate, setters })
  }

  const finishExit = useCallback(() => {
    setPhase('off')
  }, [])

  return {
    mounted: phase !== 'off',
    exiting: phase === 'out',
    instant,
    path: shownPath,
    node: active ? children : snap.current,
    finishExit,
  }
}

function syncEnter({
  phase,
  shownPath,
  pathname,
  animate,
  setters,
}: {
  phase: PushPhase
  shownPath: string
  pathname: string
  animate: boolean
  setters: LayerSetters
}) {
  if (phase === 'on' && shownPath === pathname) {
    return
  }
  setters.setPhase('on')
  setters.setShownPath(pathname)
  setters.setInstant(!animate)
}

function syncExit({
  phase,
  animate,
  setters,
}: {
  phase: PushPhase
  animate: boolean
  setters: LayerSetters
}) {
  if (phase !== 'on') {
    return
  }
  if (!animate) {
    setters.setPhase('off')
    return
  }
  setters.setPhase('out')
  setters.setInstant(false)
}
