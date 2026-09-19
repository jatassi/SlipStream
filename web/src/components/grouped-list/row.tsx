import type { ReactNode } from 'react'

import { Link } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'

type RowTone = 'default' | 'destructive' | 'warning'

export type RowProps = {
  title: ReactNode
  subtitle?: ReactNode
  leading?: ReactNode
  trailing?: ReactNode
  chevron?: boolean
  href?: string
  onClick?: () => void
  className?: string
  tone?: RowTone
}

const TONE: Record<RowTone, string> = {
  default: 'text-foreground',
  destructive: 'text-destructive',
  warning: 'text-amber-400',
}

function RowBody({
  title,
  subtitle,
  leading,
  trailing,
  chevron,
  tone,
}: Omit<RowProps, 'href' | 'onClick' | 'className'> & { tone: RowTone }) {
  return (
    <>
      {leading !== undefined && <div className="flex shrink-0 items-center">{leading}</div>}
      <div className="min-w-0 flex-1 text-left">
        <div className={cn('truncate text-body font-medium', TONE[tone])}>{title}</div>
        {subtitle !== undefined && (
          <div className="text-footnote mt-0.5 truncate text-muted-foreground">{subtitle}</div>
        )}
      </div>
      {trailing !== undefined && (
        <div className="text-body flex shrink-0 items-center text-muted-foreground">{trailing}</div>
      )}
      {chevron ? <ChevronRight className="size-4 shrink-0 text-muted-foreground/60" /> : null}
    </>
  )
}

export function Row({
  title,
  subtitle,
  leading,
  trailing,
  chevron = false,
  href,
  onClick,
  className,
  tone = 'default',
}: RowProps) {
  const body = (
    <RowBody
      title={title}
      subtitle={subtitle}
      leading={leading}
      trailing={trailing}
      chevron={chevron}
      tone={tone}
    />
  )
  const classes = cn(
    'flex min-h-tap w-full items-center gap-3 px-4 py-2.5',
    'focus-visible:ring-ring outline-none focus-visible:ring-[3px]',
    className,
  )

  if (href !== undefined) {
    return (
      <Link to={href} className={cn(classes, 'press-row')}>
        {body}
      </Link>
    )
  }

  if (onClick) {
    return (
      <button type="button" onClick={onClick} className={cn(classes, 'press-row')}>
        {body}
      </button>
    )
  }

  return <div className={classes}>{body}</div>
}
