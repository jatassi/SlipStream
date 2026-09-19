import type { ReactNode } from 'react'
import { useEffect, useRef, useState } from 'react'

import type { DragState } from './use-sheet-drag'
import { useSheetDrag } from './use-sheet-drag'

type SheetProps = {
  onClose: () => void
  header: ReactNode
  children: ReactNode
}

type Phase = 'enter' | 'open' | 'closing'

function panelTransform(drag: DragState | null, phase: Phase): string {
  if (drag !== null) {
    return `translateY(${drag.offset}px)`
  }
  return phase === 'open' ? 'translateY(0)' : 'translateY(100%)'
}

function scrimOpacity(drag: DragState | null, phase: Phase): number {
  if (drag !== null) {
    return Math.max(0, 1 - drag.ratio)
  }
  return phase === 'open' ? 1 : 0
}

export function Sheet({ onClose, header, children }: SheetProps) {
  const panelRef = useRef<HTMLDivElement>(null)
  const [phase, setPhase] = useState<Phase>('enter')
  const dismiss = () => setPhase('closing')
  const { drag, handlers } = useSheetDrag(panelRef, dismiss)

  useEffect(() => {
    const raf = requestAnimationFrame(() => setPhase('open'))
    return () => cancelAnimationFrame(raf)
  }, [])

  return (
    <div className="cine-sheet-root">
      <button type="button" aria-label="Dismiss" className="cine-scrim w-full" style={{ opacity: scrimOpacity(drag, phase) }} onClick={dismiss} />
      <div
        ref={panelRef}
        className="cine-sheet flex flex-col"
        data-animating={drag === null ? '' : undefined}
        style={{ transform: panelTransform(drag, phase) }}
        onTransitionEnd={(e) => {
          if (phase === 'closing' && e.target === e.currentTarget) {
            onClose()
          }
        }}
      >
        <div className="cine-sheet-handle-zone shrink-0" {...handlers}>
          {header}
        </div>
        <div className="m-scroll min-h-0 flex-1">{children}</div>
      </div>
    </div>
  )
}
