import { useLayoutEffect, useRef, useState } from 'react'

import { afterPaint } from '../shared/after-paint'

type PickerProps = {
  names: string[]
  active: number
  onSelect: (index: number) => void
  onReplay: () => void
}

function useHighlight(active: number, names: string[]) {
  const listRef = useRef<HTMLElement>(null)
  const highlightRef = useRef<HTMLSpanElement>(null)
  const [ready, setReady] = useState(false)

  useLayoutEffect(() => {
    const move = () => {
      const items = listRef.current?.querySelectorAll<HTMLButtonElement>(
        '.proto-picker-item:not(.proto-picker-replay)',
      )
      const el = items?.[active]
      const highlight = highlightRef.current
      if (!el || !highlight) {
        return
      }
      highlight.style.width = `${el.offsetWidth}px`
      highlight.style.transform = `translateX(${el.offsetLeft}px)`
    }
    move()
    globalThis.addEventListener('resize', move)
    return () => globalThis.removeEventListener('resize', move)
  }, [active, names])

  useLayoutEffect(() => afterPaint(() => setReady(true)), [])

  return { listRef, highlightRef, ready }
}

export function Picker({ names, active, onSelect, onReplay }: PickerProps) {
  const { listRef, highlightRef, ready } = useHighlight(active, names)

  return (
    <nav
      ref={listRef}
      className="proto-picker"
      aria-label="Prototype variants"
      data-position="top"
      data-ready={ready ? '' : undefined}
    >
      <span ref={highlightRef} className="proto-picker-highlight" aria-hidden="true" />
      {names.map((name, index) => (
        <button
          key={name}
          type="button"
          className="proto-picker-item"
          data-active={index === active ? '' : undefined}
          aria-current={index === active ? 'true' : undefined}
          onClick={() => onSelect(index)}
        >
          {name}
        </button>
      ))}
      <span className="proto-picker-divider" aria-hidden="true" />
      <button
        type="button"
        className="proto-picker-item proto-picker-replay"
        aria-label="Replay animation (R)"
        onClick={onReplay}
      >
        ↻
      </button>
    </nav>
  )
}
