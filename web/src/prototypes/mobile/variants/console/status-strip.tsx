import { Film } from 'lucide-react'

import { formatSpeed } from '../../shared/format'
import { useMobileState } from '../../shared/mobile-state-context'
import { HEALTH } from '../../shared/mock-data'

export function StatusStrip() {
  const { queue } = useMobileState()
  const active = queue.filter((q) => q.state === 'downloading')
  const speed = active.reduce((s, q) => s + q.speedMbps, 0)

  return (
    <div className="m-material m-safe-top absolute inset-x-0 top-0 z-20 shadow-[0_1px_0_var(--material-edge)]">
      <div className="flex h-10 items-center justify-between px-3">
        <div className="flex items-center gap-2">
          <span className="bg-media-gradient flex size-5 items-center justify-center rounded-[5px] text-white"><Film className="size-3" /></span>
          <span className="font-mono text-[12px] font-semibold tracking-[0.06em]">SLIPSTREAM</span>
        </div>
        <div className="m-nums flex items-center gap-3 font-mono text-[11px]">
          <span className="flex items-center gap-1.5 text-amber-400"><span className="con-led" />{HEALTH.length}</span>
          <span className="flex items-center gap-1.5 text-movie-400"><span className="con-led" />{active.length}↓ {formatSpeed(speed)}</span>
          <span className="flex items-center gap-1.5 text-emerald-400"><span className="con-led" />ws</span>
        </div>
      </div>
    </div>
  )
}
