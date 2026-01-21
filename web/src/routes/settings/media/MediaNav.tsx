import { Link, useRouterState } from '@tanstack/react-router'
import { FolderOpen, Sliders, Layers, FileInput } from 'lucide-react'
import { cn } from '@/lib/utils'

const mediaNavItems = [
  { title: 'Root Folders', href: '/settings/media/root-folders', icon: FolderOpen },
  { title: 'Quality Profiles', href: '/settings/media/quality-profiles', icon: Sliders },
  { title: 'Version Slots', href: '/settings/media/version-slots', icon: Layers },
  { title: 'Import & Naming', href: '/settings/media/file-naming', icon: FileInput },
]

export function MediaNav() {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <nav className="flex gap-1 border-b mb-6">
      {mediaNavItems.map((item) => {
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
