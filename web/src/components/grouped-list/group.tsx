import { type ReactNode, useId } from 'react'

import { cn } from '@/lib/utils'

type GroupProps = {
  header?: string
  footer?: ReactNode
  action?: ReactNode
  children: ReactNode
  className?: string
  inset?: boolean
}

export function Group({ header, footer, action, children, className, inset = true }: GroupProps) {
  const headingId = useId()
  return (
    <section
      aria-labelledby={header === undefined ? undefined : headingId}
      className={cn('mb-7', inset && 'px-screen', className)}
    >
      {header !== undefined && (
        <div className="flex items-baseline justify-between px-3 pb-2">
          <h2
            id={headingId}
            className="text-footnote font-semibold tracking-wide text-muted-foreground uppercase"
          >
            {header}
          </h2>
          {action}
        </div>
      )}
      <div className="divide-y divide-border/70 overflow-hidden rounded-card bg-card">{children}</div>
      {footer !== undefined && (
        <div className="text-footnote px-3 pt-2 text-muted-foreground">{footer}</div>
      )}
    </section>
  )
}
