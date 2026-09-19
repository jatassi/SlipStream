import type { PointerEvent, RefObject } from 'react'
import { useRef, useState } from 'react'

const DISMISS_RATIO = 0.35
const DISMISS_VELOCITY = 0.6

type Drag = { startY: number; startOffset: number; lastY: number; lastT: number; velocity: number; height: number }

export type DragState = { offset: number; ratio: number }

function currentTranslateY(el: HTMLElement): number {
  return new DOMMatrixReadOnly(getComputedStyle(el).transform).m42
}

// Real things slow before they stop: resistance grows the further past the bound the finger goes
function rubberband(overshoot: number, dimension: number, constant = 0.55): number {
  return (overshoot * dimension * constant) / (dimension + constant * Math.abs(overshoot))
}

export function useSheetDrag(panelRef: RefObject<HTMLDivElement | null>, onDismiss: () => void) {
  const dragRef = useRef<Drag | null>(null)
  const [drag, setDrag] = useState<DragState | null>(null)

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    const panel = panelRef.current
    if (!panel) {
      return
    }
    e.currentTarget.setPointerCapture(e.pointerId)
    // Start from the presentation value so a grab mid-animation never jumps
    const startOffset = currentTranslateY(panel)
    const height = panel.offsetHeight
    dragRef.current = { startY: e.clientY, startOffset, lastY: e.clientY, lastT: e.timeStamp, velocity: 0, height }
    setDrag({ offset: startOffset, ratio: startOffset / height })
  }

  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const current = dragRef.current
    if (!current) {
      return
    }
    const dt = Math.max(1, e.timeStamp - current.lastT)
    current.velocity = (e.clientY - current.lastY) / dt
    current.lastY = e.clientY
    current.lastT = e.timeStamp
    const raw = current.startOffset + (e.clientY - current.startY)
    const offset = raw < 0 ? rubberband(raw, current.height) : raw
    setDrag({ offset, ratio: Math.max(0, offset / current.height) })
  }

  const onPointerUp = () => {
    const current = dragRef.current
    dragRef.current = null
    if (!current || drag === null) {
      return
    }
    // A quick flick dismisses regardless of distance
    const shouldClose = drag.ratio > DISMISS_RATIO || current.velocity > DISMISS_VELOCITY
    setDrag(null)
    if (shouldClose) {
      onDismiss()
    }
  }

  return { drag, handlers: { onPointerDown, onPointerMove, onPointerUp, onPointerCancel: onPointerUp } }
}
