import { useSyncExternalStore } from 'react'

export type ViewportShell = 'phone' | 'wide'

const WIDE_QUERY = '(min-width: 768px)'

function subscribe(onChange: () => void) {
  const mql = globalThis.matchMedia(WIDE_QUERY)
  mql.addEventListener('change', onChange)
  return () => {
    mql.removeEventListener('change', onChange)
  }
}

function snapshot(): ViewportShell {
  return globalThis.matchMedia(WIDE_QUERY).matches ? 'wide' : 'phone'
}

function serverSnapshot(): ViewportShell {
  return 'wide'
}

export function useViewport(): ViewportShell {
  return useSyncExternalStore(subscribe, snapshot, serverSnapshot)
}
