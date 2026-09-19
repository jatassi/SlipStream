import type { ReactNode } from 'react'
import { useEffect, useState } from 'react'

import { BatteryFull, Signal, Wifi } from 'lucide-react'

const FRAME_W = 390
const FRAME_H = 844
const STAGE_MARGIN_Y = 120
const STAGE_MARGIN_X = 32

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => globalThis.matchMedia(query).matches)
  useEffect(() => {
    const mq = globalThis.matchMedia(query)
    const onChange = () => setMatches(mq.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [query])
  return matches
}

function computeScale(): number {
  const byHeight = (globalThis.innerHeight - STAGE_MARGIN_Y) / FRAME_H
  const byWidth = (globalThis.innerWidth - STAGE_MARGIN_X) / FRAME_W
  return Math.min(1, byHeight, byWidth)
}

function useFrameScale(): number {
  const [scale, setScale] = useState(computeScale)
  useEffect(() => {
    const onResize = () => setScale(computeScale())
    globalThis.addEventListener('resize', onResize)
    return () => globalThis.removeEventListener('resize', onResize)
  }, [])
  return scale
}

function StatusBar() {
  return (
    <div className="m-status-bar" aria-hidden="true">
      <span className="m-nums">9:41</span>
      <span className="flex items-center gap-1.5">
        <Signal className="size-4" strokeWidth={2.5} />
        <Wifi className="size-4" strokeWidth={2.5} />
        <BatteryFull className="size-5" strokeWidth={2} />
      </span>
    </div>
  )
}

export function PhoneFrame({ children }: { children: ReactNode }) {
  const isRealPhone = useMediaQuery('(max-width: 520px)')
  const scale = useFrameScale()

  if (isRealPhone) {
    return <div className="m-device-real">{children}</div>
  }

  return (
    <div className="m-stage">
      <div className="m-phone" style={{ transform: `scale(${scale})` }}>
        <div className="m-phone-screen">
          {children}
          <StatusBar />
          <div className="m-island" aria-hidden="true" />
          <div className="m-home-indicator" aria-hidden="true" />
        </div>
      </div>
    </div>
  )
}
