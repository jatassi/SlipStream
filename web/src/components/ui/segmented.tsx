import type { KeyboardEvent } from 'react'

import { cn } from '@/lib/utils'

import './segmented.css'

export type SegmentedOption<T extends string> = {
  value: T
  label: string
}

type SegmentedProps<T extends string> = {
  label: string
  value: T
  options: SegmentedOption<T>[]
  onChange: (value: T) => void
  className?: string
}

const STEP: Partial<Record<string, number>> = {
  ArrowRight: 1,
  ArrowDown: 1,
  ArrowLeft: -1,
  ArrowUp: -1,
}

function keyTarget(key: string, index: number, count: number): number | null {
  const step = STEP[key]
  if (step !== undefined) {
    return (index + step + count) % count
  }
  if (key === 'Home') {
    return 0
  }
  if (key === 'End') {
    return count - 1
  }
  return null
}

function focusSegment(group: HTMLElement, position: number) {
  const segments = [...group.querySelectorAll<HTMLButtonElement>('[role="radio"]')]
  segments.at(position)?.focus()
}

function Thumb({ index, count }: { index: number; count: number }) {
  return (
    <span
      aria-hidden="true"
      className="segmented-thumb bg-card absolute top-0.5 bottom-0.5 left-0.5 rounded-[7px] shadow-[0_1px_3px_rgba(0,0,0,0.4),0_0_0_0.5px_rgba(255,255,255,0.06)]"
      style={{
        width: `calc((100% - 4px) / ${count})`,
        transform: `translateX(${index * 100}%)`,
      }}
    />
  )
}

function SegmentButton<T extends string>({
  option,
  selected,
  onSelect,
}: {
  option: SegmentedOption<T>
  selected: boolean
  onSelect: (value: T) => void
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      tabIndex={selected ? 0 : -1}
      onClick={() => {
        onSelect(option.value)
      }}
      className={cn(
        'text-footnote focus-visible:ring-ring relative z-10 rounded-[7px] font-semibold transition-colors duration-150 outline-none focus-visible:ring-[3px]',
        selected ? 'text-foreground' : 'text-muted-foreground',
      )}
    >
      {option.label}
    </button>
  )
}

export function Segmented<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
}: SegmentedProps<T>) {
  const index = Math.max(
    0,
    options.findIndex((option) => option.value === value),
  )

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const target = keyTarget(event.key, index, options.length)
    if (target === null) {
      return
    }
    event.preventDefault()
    onChange(options[target].value)
    focusSegment(event.currentTarget, target)
  }

  return (
    <div
      role="radiogroup"
      aria-label={label}
      tabIndex={-1}
      onKeyDown={onKeyDown}
      className={cn('relative grid h-9 rounded-[9px] bg-foreground/8 p-0.5', className)}
      style={{ gridTemplateColumns: `repeat(${options.length}, minmax(0, 1fr))` }}
    >
      <Thumb index={index} count={options.length} />
      {options.map((option) => (
        <SegmentButton
          key={option.value}
          option={option}
          selected={option.value === value}
          onSelect={onChange}
        />
      ))}
    </div>
  )
}
