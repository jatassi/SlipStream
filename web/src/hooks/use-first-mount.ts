import { useEffect, useState } from 'react'

const mounted = new Set<string>()

/** Longest entrance the motion policy allows: --dur-base plus the stagger cap. */
const ENTRANCE_MS = 500

/**
 * True only while `key` is mounting for the first time in this page session.
 * Entrance animations gate on it, so they play once and are not replayed when a
 * tab is returned to or a screen is reached with the back control.
 */
export function useFirstMount(key: string): boolean {
  const [first, setFirst] = useState(() => !mounted.has(key))

  useEffect(() => {
    mounted.add(key)
    if (!first) {
      return
    }
    const played = setTimeout(() => setFirst(false), ENTRANCE_MS)
    return () => clearTimeout(played)
  }, [first, key])

  return first
}
