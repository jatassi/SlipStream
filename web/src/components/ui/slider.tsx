import type * as React from 'react'

import { cn } from '@/lib/utils'

type SliderProps = {
  value?: number[]
  defaultValue?: number[]
  onValueChange?: (value: number[]) => void
  min?: number
  max?: number
  step?: number
  disabled?: boolean
  className?: string
}

function Slider({ value, defaultValue, onValueChange, min = 0, max = 100, step = 1, disabled, className }: SliderProps) {
  const current = value?.[0] ?? defaultValue?.[0] ?? min
  const percent = ((current - min) / (max - min)) * 100

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onValueChange?.([Number(e.target.value)])
  }

  return (
    <div
      data-slot="slider"
      className={cn('relative flex w-full touch-none items-center select-none', disabled && 'opacity-50', className)}
    >
      <div className="relative h-1 w-full rounded-full bg-muted" data-slot="slider-track">
        <div
          data-slot="slider-range"
          className="absolute h-full rounded-full bg-primary"
          style={{ width: `${percent}%` }}
        />
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={current}
        disabled={disabled}
        onChange={handleChange}
        className="absolute inset-0 m-0 h-full w-full cursor-pointer appearance-none bg-transparent opacity-0"
      />
      <div
        data-slot="slider-thumb"
        className="border-ring ring-ring/50 pointer-events-none absolute top-1/2 size-3 -translate-y-1/2 rounded-full border bg-white transition-shadow"
        style={{ left: `calc(${percent}% - 6px)` }}
      />
    </div>
  )
}

export { Slider }
export type { SliderProps }
