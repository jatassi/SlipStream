import { useCallback, useState } from 'react'

type PushPhase = 'off' | 'on' | 'out'

type PushLayer = {
  mounted: boolean
  exiting: boolean
  instant: boolean
  path: string
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
  animate,
}: {
  active: boolean
  pathname: string
  animate: boolean
}): PushLayer {
  const [phase, setPhase] = useState<PushPhase>(active ? 'on' : 'off')
  const [shownPath, setShownPath] = useState(pathname)
  const [instant, setInstant] = useState(!animate)
  const setters: LayerSetters = { setPhase, setShownPath, setInstant }

  if (active) {
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
