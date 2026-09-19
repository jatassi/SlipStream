import type { ReactNode } from 'react'

import { ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'

export function Group({ header, action, children, className }: { header?: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={cn('mb-7 px-screen', className)}>
      {header !== undefined && (
        <div className="flex items-baseline justify-between px-3 pb-2">
          <h2 className="text-m-footnote font-semibold tracking-wide text-muted-foreground uppercase">{header}</h2>
          {action}
        </div>
      )}
      <div className="divide-y divide-border/70 overflow-hidden rounded-card bg-card">{children}</div>
    </section>
  )
}

type RowProps = {
  title: ReactNode
  subtitle?: ReactNode
  leading?: ReactNode
  trailing?: ReactNode
  chevron?: boolean
  onClick?: () => void
  className?: string
  tone?: 'default' | 'destructive' | 'warning'
}

const TONE: Record<NonNullable<RowProps['tone']>, string> = {
  default: 'text-foreground',
  destructive: 'text-destructive',
  warning: 'text-amber-400',
}

export function Row({ title, subtitle, leading, trailing, chevron = false, onClick, className, tone = 'default' }: RowProps) {
  const body = (
    <>
      {leading !== undefined && <div className="flex shrink-0 items-center">{leading}</div>}
      <div className="min-w-0 flex-1 text-left">
        <div className={cn('truncate text-m-body font-medium', TONE[tone])}>{title}</div>
        {subtitle !== undefined && <div className="mt-0.5 truncate text-m-footnote text-muted-foreground">{subtitle}</div>}
      </div>
      {trailing !== undefined && <div className="flex shrink-0 items-center text-m-body text-muted-foreground">{trailing}</div>}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted-foreground/60" /> : null}
    </>
  )
  const classes = cn('flex min-h-tap w-full items-center gap-3 px-4 py-2.5', className)

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className={cn(classes, 'm-press-row')}>
        {body}
      </button>
    )
  }
  return <div className={classes}>{body}</div>
}

export function IconTile({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span className={cn('flex size-7 items-center justify-center rounded-[7px] text-white [&_svg]:size-4', className)}>
      {children}
    </span>
  )
}
