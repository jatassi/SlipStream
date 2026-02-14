import type { ReactNode } from 'react'

import { Link } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'

type BreadcrumbItem = {
  label: string
  href?: string
}

type PageHeaderProps = {
  title: string
  description?: ReactNode
  breadcrumbs?: BreadcrumbItem[]
  actions?: ReactNode
  className?: string
}

export function PageHeader({
  title,
  description,
  breadcrumbs,
  actions,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn('mb-6 space-y-2', className)}>
      {/* Breadcrumbs */}
      {breadcrumbs && breadcrumbs.length > 0 ? (
        <nav className="text-muted-foreground flex items-center gap-1 text-sm">
          {breadcrumbs.map((item, index) => (
            <span key={index} className="flex items-center gap-1">
              {index > 0 && <ChevronRight className="size-4" />}
              {item.href ? (
                <Link to={item.href} className="hover:text-foreground transition-colors">
                  {item.label}
                </Link>
              ) : (
                <span className="text-foreground">{item.label}</span>
              )}
            </span>
          ))}
        </nav>
      ) : null}

      {/* Title and actions */}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
          {description ? <p className="text-muted-foreground">{description}</p> : null}
        </div>
        {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
      </div>
    </div>
  )
}
