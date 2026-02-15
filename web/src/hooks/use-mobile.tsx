import { useSyncExternalStore } from 'react'

const MOBILE_BREAKPOINT = 768

function getIsMobile(): boolean {
  return globalThis.window.innerWidth < MOBILE_BREAKPOINT
}

function subscribe(callback: () => void): () => void {
  const mql = globalThis.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`)
  mql.addEventListener('change', callback)
  return () => mql.removeEventListener('change', callback)
}

export function useIsMobile() {
  return useSyncExternalStore(subscribe, getIsMobile, () => false)
}
