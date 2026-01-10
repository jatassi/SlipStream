import { Link, useRouterState } from '@tanstack/react-router'
import {
  LayoutDashboard,
  Film,
  Tv,
  Calendar,
  Activity,
  History,
  Settings,
  FolderOpen,
  Sliders,
  Rss,
  Download,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Clock,
  Search,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { useUIStore } from '@/stores'
import { useMissingCounts } from '@/hooks'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { ProgressContainer } from '@/components/progress'

interface NavItem {
  title: string
  href: string
  icon: React.ElementType
}

interface CollapsibleNavGroup {
  id: string
  title: string
  icon: React.ElementType
  items: NavItem[]
}

const mainNavItems: NavItem[] = [
  { title: 'Dashboard', href: '/', icon: LayoutDashboard },
  { title: 'Movies', href: '/movies', icon: Film },
  { title: 'Series', href: '/series', icon: Tv },
  { title: 'Calendar', href: '/calendar', icon: Calendar },
  { title: 'Missing', href: '/missing', icon: Search },
]

const activityGroup: CollapsibleNavGroup = {
  id: 'activity',
  title: 'Activity',
  icon: Activity,
  items: [
    { title: 'Downloads', href: '/activity', icon: Download },
    { title: 'History', href: '/activity/history', icon: History },
  ],
}

const settingsGroup: CollapsibleNavGroup = {
  id: 'settings',
  title: 'Settings',
  icon: Settings,
  items: [
    { title: 'General', href: '/settings', icon: Settings },
    { title: 'Quality Profiles', href: '/settings/profiles', icon: Sliders },
    { title: 'Root Folders', href: '/settings/rootfolders', icon: FolderOpen },
    { title: 'Indexers', href: '/settings/indexers', icon: Rss },
    { title: 'Download Clients', href: '/settings/downloadclients', icon: Download },
  ],
}

const systemGroup: CollapsibleNavGroup = {
  id: 'system',
  title: 'System',
  icon: Clock,
  items: [
    { title: 'Scheduled Tasks', href: '/system/tasks', icon: Clock },
  ],
}

function NavLink({
  item,
  collapsed,
  indented = false,
  badge,
}: {
  item: NavItem
  collapsed: boolean
  indented?: boolean
  badge?: React.ReactNode
}) {
  const router = useRouterState()
  const isActive = router.location.pathname === item.href

  const linkContent = (
    <>
      <item.icon className="size-4 shrink-0" />
      {!collapsed && (
        <>
          <span className="flex-1">{item.title}</span>
          {badge}
        </>
      )}
    </>
  )

  const linkClassName = cn(
    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
    'hover:bg-accent hover:text-accent-foreground',
    isActive && 'bg-accent text-accent-foreground',
    collapsed && 'justify-center px-2',
    indented && !collapsed && 'ml-4 border-l border-border pl-4'
  )

  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger
          render={<Link to={item.href} className={linkClassName} />}
        >
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

function MissingBadge() {
  const { data: counts } = useMissingCounts()

  if (!counts || (counts.movies === 0 && counts.episodes === 0)) {
    return null
  }

  return (
    <span className="flex items-center gap-1 text-xs text-muted-foreground">
      {counts.movies > 0 && (
        <span className="flex items-center gap-0.5">
          <Film className="size-3" />
          {counts.movies}
        </span>
      )}
      {counts.episodes > 0 && (
        <span className="flex items-center gap-0.5">
          <Tv className="size-3" />
          {counts.episodes}
        </span>
      )}
    </span>
  )
}

function NavSection({ items, collapsed }: { items: NavItem[]; collapsed: boolean }) {
  return (
    <div className="space-y-1">
      {items.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          collapsed={collapsed}
          badge={item.href === '/missing' ? <MissingBadge /> : undefined}
        />
      ))}
    </div>
  )
}

function CollapsibleNavSection({
  group,
  collapsed: sidebarCollapsed,
}: {
  group: CollapsibleNavGroup
  collapsed: boolean
}) {
  const router = useRouterState()
  const { expandedMenus, toggleMenu } = useUIStore()

  const isExpanded = expandedMenus[group.id] ?? false
  const isAnyChildActive = group.items.some(
    (item) => router.location.pathname === item.href
  )

  // When sidebar is collapsed, show a tooltip with submenu items
  if (sidebarCollapsed) {
    return (
      <Tooltip>
        <TooltipTrigger
          className={cn(
            'flex w-full items-center justify-center rounded-md px-2 py-2 text-sm font-medium transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            isAnyChildActive && 'bg-accent text-accent-foreground'
          )}
        >
          <group.icon className="size-5 shrink-0" />
        </TooltipTrigger>
        <TooltipContent side="right" className="flex flex-col gap-1 p-2">
          <span className="mb-1 text-xs font-semibold">{group.title}</span>
          {group.items.map((item) => (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                'flex items-center gap-2 rounded px-2 py-1 text-sm hover:bg-accent',
                router.location.pathname === item.href && 'bg-accent'
              )}
            >
              <item.icon className="size-3" />
              {item.title}
            </Link>
          ))}
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Collapsible open={isExpanded} onOpenChange={() => toggleMenu(group.id)}>
      <CollapsibleTrigger
        className={cn(
          'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
          'hover:bg-accent hover:text-accent-foreground',
          isAnyChildActive && 'text-accent-foreground'
        )}
      >
        <group.icon className="size-5 shrink-0" />
        <span className="flex-1 text-left">{group.title}</span>
        <ChevronDown
          className={cn(
            'size-4 shrink-0 transition-transform duration-200',
            isExpanded && 'rotate-180'
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="overflow-hidden data-[ending-style]:animate-collapse-up data-[starting-style]:animate-collapse-down">
        <div className="mt-1 space-y-1">
          {group.items.map((item) => (
            <NavLink key={item.href} item={item} collapsed={false} indented />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function Sidebar() {
  const { sidebarCollapsed, toggleSidebar } = useUIStore()

  return (
    <TooltipProvider delay={0}>
      <aside
        className={cn(
          'flex h-screen flex-col border-r border-border bg-card transition-all duration-300',
          sidebarCollapsed ? 'w-16' : 'w-64'
        )}
      >
        {/* Logo */}
        <div
          className={cn(
            'flex h-14 items-center border-b border-border px-4',
            sidebarCollapsed && 'justify-center px-2'
          )}
        >
          <Link to="/" className="flex items-center gap-2">
            <div className="flex size-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <Film className="size-5" />
            </div>
            {!sidebarCollapsed && (
              <span className="text-lg font-semibold">SlipStream</span>
            )}
          </Link>
        </div>

        {/* Navigation */}
        <ScrollArea className="flex-1 px-3 py-4">
          <nav className="space-y-4">
            {/* Main navigation */}
            <NavSection items={mainNavItems} collapsed={sidebarCollapsed} />

            {/* Divider */}
            <div className="h-px bg-border" />

            {/* Activity collapsible menu */}
            <CollapsibleNavSection
              group={activityGroup}
              collapsed={sidebarCollapsed}
            />

            {/* Settings collapsible menu */}
            <CollapsibleNavSection
              group={settingsGroup}
              collapsed={sidebarCollapsed}
            />

            {/* System collapsible menu */}
            <CollapsibleNavSection
              group={systemGroup}
              collapsed={sidebarCollapsed}
            />
          </nav>
        </ScrollArea>

        {/* Progress indicators */}
        <ProgressContainer
          collapsed={sidebarCollapsed}
          className="border-t border-border"
        />

        {/* Collapse toggle */}
        <div className="border-t border-border p-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleSidebar}
            className={cn('w-full', sidebarCollapsed && 'px-2')}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="size-4" />
            ) : (
              <>
                <ChevronLeft className="size-4" />
                <span className="ml-2">Collapse</span>
              </>
            )}
          </Button>
        </div>
      </aside>
    </TooltipProvider>
  )
}
