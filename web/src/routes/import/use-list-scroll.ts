import { useEffect, useLayoutEffect, useRef } from 'react'

// Entering a folder replaces the list inside one scroll container, so each
// folder's offset is remembered here and restored when its list comes back.
export function useListScroll(path: string, ready: boolean) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const offsets = useRef(new Map<string, number>())
  const pathRef = useRef(path)
  const restoredFor = useRef<string | null>(null)

  useEffect(() => {
    const element = scrollRef.current
    if (element === null) {
      return
    }
    const onScroll = () => {
      offsets.current.set(pathRef.current, element.scrollTop)
    }
    element.addEventListener('scroll', onScroll, { passive: true })
    return () => {
      element.removeEventListener('scroll', onScroll)
    }
  }, [])

  useLayoutEffect(() => {
    pathRef.current = path
    const element = scrollRef.current
    if (element === null || !ready || restoredFor.current === path) {
      return
    }
    restoredFor.current = path
    element.scrollTop = offsets.current.get(path) ?? 0
  }, [path, ready])

  return scrollRef
}
