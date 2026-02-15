import { Link, useRouterState } from '@tanstack/react-router'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { NavItem } from './sidebar-types'

type NavLinkProps = {
  item: NavItem
  collapsed: boolean
  indented?: boolean
  badge?: React.ReactNode
}

function getIconClassName(theme: NavItem['theme'], isActive: boolean): string {
  return cn(
    'size-4 shrink-0 transition-colors',
    theme === 'movie' && 'text-movie-500',
    theme === 'tv' && 'text-tv-500',
    isActive && theme === 'movie' && 'text-movie-400',
    isActive && theme === 'tv' && 'text-tv-400',
  )
}

function getThemeClassName(theme: NavItem['theme'], isActive: boolean): string {
  if (theme === 'movie') {
    return cn(
      'hover:bg-movie-500/10 hover:text-foreground',
      isActive && 'bg-movie-500/15 text-foreground border-l-movie-500',
    )
  }
  if (theme === 'tv') {
    return cn(
      'hover:bg-tv-500/10 hover:text-foreground',
      isActive && 'bg-tv-500/15 text-foreground border-l-tv-500',
    )
  }
  return cn(
    'hover:bg-accent hover:text-accent-foreground',
    isActive && 'bg-accent text-accent-foreground',
  )
}

function getLinkClassName(opts: { collapsed: boolean; indented: boolean; theme: NavItem['theme']; isActive: boolean }): string {
  return cn(
    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all border-l-2 border-transparent',
    opts.collapsed && 'justify-center px-2 border-l-0',
    opts.indented && !opts.collapsed && 'ml-4 border-l border-border pl-4',
    getThemeClassName(opts.theme, opts.isActive),
  )
}

export function NavLink({ item, collapsed, indented = false, badge }: NavLinkProps) {
  const router = useRouterState()
  const isActive =
    router.location.pathname === item.href || router.location.pathname.startsWith(`${item.href}/`)

  const iconClassName = getIconClassName(item.theme, isActive)
  const linkClassName = getLinkClassName({ collapsed, indented, theme: item.theme, isActive })

  const linkContent = (
    <>
      <item.icon className={iconClassName} />
      {!collapsed && (
        <>
          <span className="flex-1">{item.title}</span>
          {badge}
        </>
      )}
    </>
  )

  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger render={<Link to={item.href} className={linkClassName} />}>
          {linkContent}
        </TooltipTrigger>
        <TooltipContent side="right">
          <div className="flex items-center gap-2">
            {item.title}
            {badge}
          </div>
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Link to={item.href} className={linkClassName}>
      {linkContent}
    </Link>
  )
}
