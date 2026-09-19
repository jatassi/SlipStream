import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

import { kindColor, STATUS_LABEL, statusColor } from './format'
import type { MediaKind, MediaStatus } from './types'

export function StatusDot({ status, className }: { status: MediaStatus; className?: string }) {
  return (
    <span
      className={cn('inline-block size-1.5 shrink-0 rounded-full', className)}
      style={{ background: statusColor(status) }}
      aria-label={STATUS_LABEL[status]}
    />
  )
}

export function StatusPill({ status, className }: { status: MediaStatus; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex h-6 items-center gap-1.5 rounded-full px-2 text-m-caption font-semibold',
        className,
      )}
      style={{
        color: statusColor(status),
        background: `color-mix(in oklch, ${statusColor(status)} 14%, transparent)`,
      }}
    >
      <StatusDot status={status} />
      {STATUS_LABEL[status]}
    </span>
  )
}

type ProgressLineProps = {
  value: number
  kind: MediaKind
  className?: string
  height?: string
  muted?: boolean
}

export function ProgressLine({ value, kind, className, height = 'h-1', muted = false }: ProgressLineProps) {
  return (
    <div
      className={cn('w-full overflow-hidden rounded-full bg-foreground/10', height, className)}
      role="progressbar"
      aria-valuenow={Math.round(value)}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <div
        className="h-full rounded-full transition-[width] duration-700 ease-linear"
        style={{ width: `${value}%`, background: muted ? 'var(--muted-foreground)' : kindColor(kind) }}
      />
    </div>
  )
}

export function KindMark({ kind, className }: { kind: MediaKind; className?: string }) {
  return (
    <span
      className={cn('inline-block size-2 shrink-0 rounded-[3px]', className)}
      style={{ background: kindColor(kind) }}
      aria-hidden="true"
    />
  )
}

export function Chip({ children, active = false, onClick, className }: { children: ReactNode; active?: boolean; onClick?: () => void; className?: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'm-press inline-flex h-8 shrink-0 items-center gap-1.5 rounded-full px-3 text-m-footnote font-medium whitespace-nowrap',
        active ? 'bg-foreground text-background' : 'bg-foreground/8 text-foreground/80',
        className,
      )}
    >
      {children}
    </button>
  )
}
