import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SettingsSectionProps {
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
          {Icon && (
            <Icon className="size-8 shrink-0" />
          )}
          <div className="space-y-0.5">
            <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
            {description && (
              <p className="text-sm text-muted-foreground">{description}</p>
            )}
          </div>
        </div>
        {actions && <div className="shrink-0">{actions}</div>}
      </div>
      <div>{children}</div>
    </section>
  )
}
