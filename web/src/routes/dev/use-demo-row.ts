import { useCallback, useEffect, useRef, useState } from 'react'

import type { DemoState } from './controls-types'

export function useDemoRow() {
  const [state, setState] = useState<DemoState>({ type: 'default' })
  const [monitored, setMonitored] = useState(true)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const cleanup = useCallback(() => {
    if (timerRef.current) { clearTimeout(timerRef.current) }
    if (intervalRef.current) { clearInterval(intervalRef.current) }
    timerRef.current = null
    intervalRef.current = null
  }, [])

  useEffect(() => cleanup, [cleanup])

  const tickProgress = useCallback((pct: number) => {
    if (pct >= 100) {
      if (intervalRef.current) { clearInterval(intervalRef.current) }
      setState({ type: 'completed' })
      timerRef.current = setTimeout(() => setState({ type: 'default' }), 2500)
    } else {
      setState({ type: 'progress', percent: pct })
    }
  }, [])

  const startProgress = useCallback(() => {
    let pct = 0
    setState({ type: 'progress', percent: 0 })
    intervalRef.current = setInterval(() => { pct += 2; tickProgress(pct) }, 100)
  }, [tickProgress])

  const runAutoSearch = useCallback(() => {
    cleanup()
    setState({ type: 'searching', mode: 'auto' })
    timerRef.current = setTimeout(startProgress, 2000)
  }, [cleanup, startProgress])

  const runManualSearch = useCallback(() => {
    cleanup()
    setState({ type: 'searching', mode: 'manual' })
    timerRef.current = setTimeout(() => setState({ type: 'default' }), 2000)
  }, [cleanup])

  return { state, monitored, setMonitored, runAutoSearch, runManualSearch, isDefault: state.type === 'default' }
}
