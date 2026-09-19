import type { ComponentType, ReactNode } from 'react'

import { cn } from '@/lib/utils'

export type PillTint = 'movie' | 'tv'

export type PillActionProps = {
  label: ReactNode
  tint: PillTint
  icon?: ComponentType<{ className?: string }>
  active?: boolean
  disabled?: boolean
  pressed?: boolean
  onClick?: () => void
  ariaLabel?: string
  className?: string
}

function tintColor(tint: PillTint): string {
  return tint === 'movie' ? 'var(--movie-500)' : 'var(--tv-500)'
}

export function PillAction({
  label,
  tint,
  icon: Icon,
  active = false,
  disabled = false,
  pressed,
  onClick,
  ariaLabel,
  className,
}: PillActionProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={ariaLabel}
      aria-pressed={pressed}
      style={active ? { background: tintColor(tint) } : undefined}
      className={cn(
        'press text-footnote flex h-tap min-w-0 flex-1 items-center justify-center gap-1.5 rounded-full font-semibold',
        'focus-visible:ring-ring outline-none focus-visible:ring-[3px]',
        'disabled:opacity-60',
        active ? 'text-background' : 'bg-foreground/8 text-foreground',
        className,
      )}
    >
      {Icon === undefined ? null : <Icon className="size-4 shrink-0" />}
      <span className="truncate">{label}</span>
    </button>
  )
}

export function PillRow({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('flex gap-2', className)}>{children}</div>
}
