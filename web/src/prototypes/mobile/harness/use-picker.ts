import { useCallback, useEffect, useState } from 'react'

function readInitial(count: number): number {
  const raw = Number.parseInt(new URLSearchParams(location.search).get('v') ?? '', 10)
  if (Number.isNaN(raw) || raw < 1 || raw > count) {
    return 0
  }
  return raw - 1
}

function writeParam(index: number) {
  const url = new URL(location.href)
  url.searchParams.set('v', String(index + 1))
  history.replaceState(null, '', url)
}

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  return /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName) || target.isContentEditable
}

function shouldIgnore(e: KeyboardEvent): boolean {
  return isTypingTarget(e.target) || e.metaKey || e.ctrlKey || e.altKey
}

export function usePicker(count: number) {
  const [active, setActive] = useState(() => readInitial(count))
  const [mountKey, setMountKey] = useState(0)

  const select = useCallback(
    (index: number) => {
      if (index < 0 || index >= count) {
        return
      }
      setActive(index)
      writeParam(index)
      setMountKey((k) => k + 1)
    },
    [count],
  )

  const replay = useCallback(() => setMountKey((k) => k + 1), [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (shouldIgnore(e)) {
        return
      }
      const num = Number.parseInt(e.key, 10)
      if (num >= 1 && num <= count) {
        select(num - 1)
      } else {switch (e.key) {
 case 'ArrowRight': {
        select((active + 1) % count)
      
 break;
 }
 case 'ArrowLeft': {
        select((active - 1 + count) % count)
      
 break;
 }
 case 'r': 
 case 'R': {
        replay()
      
 break;
 }
 // No default
 }}
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [active, count, select, replay])

  return { active, mountKey, select, replay }
}
