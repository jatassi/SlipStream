import { Link, useRouterState } from '@tanstack/react-router'
import { Lock, Server } from 'lucide-react'

import { cn } from '@/lib/utils'

const systemNavItems = [
  { title: 'Server', href: '/settings/system/server', icon: Server },
  { title: 'Authentication', href: '/settings/system/authentication', icon: Lock },
]

export function SystemNav() {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <nav className="mb-6 flex gap-1 border-b">
      {systemNavItems.map((item) => {
        const isActive = currentPath === item.href
        return (
          <Link
            key={item.href}
            to={item.href}
            className={cn(
              '-mb-px flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'border-primary text-primary'
                : 'text-muted-foreground hover:text-foreground hover:border-muted-foreground/50 border-transparent',
            )}
          >
            <item.icon className="size-4" />
            {item.title}
          </Link>
        )
      })}
    </nav>
  )
}
