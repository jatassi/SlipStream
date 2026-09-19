import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

import { STATUS_LABEL, statusColor } from '../../shared/format'
import type { MediaStatus } from '../../shared/types'

export function SectionLabel({ children, trailing }: { children: ReactNode; trailing?: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between px-3 pt-5 pb-1.5">
      <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground uppercase">{children}</h2>
      {trailing !== undefined && <span className="m-nums font-mono text-[11px] text-muted-foreground">{trailing}</span>}
    </div>
  )
}

type DenseRowProps = {
  leading?: ReactNode
  title: ReactNode
  meta?: ReactNode
  trailing?: ReactNode
  onClick?: () => void
  className?: string
  children?: ReactNode
}

export function DenseRow({ leading, title, meta, trailing, onClick, className, children }: DenseRowProps) {
  const Main = onClick ? 'button' : 'div'
  return (
    <div className={cn('relative', className)}>
      <div className={cn('flex min-h-11 items-center gap-2.5 px-3 py-1.5', onClick && 'm-press-row')}>
        <Main type={onClick ? 'button' : undefined} onClick={onClick} className="flex min-w-0 flex-1 items-center gap-2.5 text-left">
          {leading !== undefined && <span className="flex shrink-0 items-center">{leading}</span>}
          <span className="min-w-0 flex-1">
            <span className="block truncate text-[14px] leading-[18px] font-medium">{title}</span>
            {meta !== undefined && <span className="m-nums block truncate text-[12px] leading-[16px] text-muted-foreground">{meta}</span>}
          </span>
        </Main>
        {trailing !== undefined && <span className="m-nums flex shrink-0 items-center gap-1.5 text-[12px] text-muted-foreground">{trailing}</span>}
      </div>
      {children}
    </div>
  )
}

export function StatusTag({ status }: { status: MediaStatus }) {
  return (
    <span className="inline-flex items-center gap-1 font-mono text-[11px] font-medium" style={{ color: statusColor(status) }}>
      <span className="con-led" />
      {STATUS_LABEL[status]}
    </span>
  )
}

export function Tag({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span className={cn('inline-flex h-[18px] items-center rounded-[4px] bg-foreground/8 px-1.5 font-mono text-[10.5px] font-medium text-foreground/80', className)}>
      {children}
    </span>
  )
}

export function InlineButton({ children, onClick, tone = 'default', className }: { children: ReactNode; onClick?: () => void; tone?: 'default' | 'danger' | 'accent'; className?: string }) {
  const tones = {
    default: 'bg-foreground/8 text-foreground',
    danger: 'bg-red-500/12 text-red-400',
    accent: 'bg-tv-500/15 text-tv-300',
  }
  return (
    <button type="button" onClick={onClick} className={cn('m-press inline-flex h-8 items-center gap-1 rounded-[7px] px-2.5 text-[12px] font-semibold', tones[tone], className)}>
      {children}
    </button>
  )
}
