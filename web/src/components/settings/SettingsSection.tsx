import type { ReactNode } from 'react'

import type { LucideIcon } from 'lucide-react'

import { cn } from '@/lib/utils'

type SettingsSectionProps = {
  id?: string
  title: string
  description?: string
  icon?: LucideIcon
  actions?: ReactNode
  children: ReactNode
  className?: string
}

export function SettingsSection({
  id,
  title,
  description,
  icon: Icon,
  actions,
  children,
  className,
}: SettingsSectionProps) {
  return (
    <section id={id} className={cn('space-y-4', className)}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {Icon ? <Icon className="size-8 shrink-0" /> : null}
          <div className="space-y-0.5">
            <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
            {description ? <p className="text-muted-foreground text-sm">{description}</p> : null}
          </div>
        </div>
        {actions ? <div className="shrink-0">{actions}</div> : null}
      </div>
      <div>{children}</div>
    </section>
  )
}
