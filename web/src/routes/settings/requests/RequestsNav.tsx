import { Link, useRouterState } from '@tanstack/react-router'
import { ListTodo, Users, Settings2 } from 'lucide-react'
import { cn } from '@/lib/utils'

const requestsNavItems = [
  { title: 'Queue', href: '/settings/requests', icon: ListTodo },
  { title: 'Users', href: '/settings/requests/users', icon: Users },
  { title: 'Settings', href: '/settings/requests/settings', icon: Settings2 },
]

export function RequestsNav() {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <nav className="flex gap-1 border-b mb-6">
      {requestsNavItems.map((item) => {
        const isActive = currentPath === item.href
        return (
          <Link
            key={item.href}
            to={item.href}
            className={cn(
              'flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
              isActive
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/50'
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
