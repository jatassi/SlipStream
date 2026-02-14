import { useEffect, useState } from 'react'

import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import {
  ArrowUpCircle,
  Bell,
  Binoculars,
  Calendar,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clock,
  Cog,
  Download,
  FileInput,
  Film,
  FolderOpen,
  HeartPulse,
  History,
  Loader2,
  LogOut,
  RotateCcw,
  ScrollText,
  Server,
  Settings,
  Tv,
  Users,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useMissingCounts, useRestart } from '@/hooks'
import { cn } from '@/lib/utils'
import { usePortalAuthStore, useUIStore } from '@/stores'

import { DownloadsNavLink } from './DownloadsNavLink'

type NavItem = {
  title: string
  href: string
  icon: React.ElementType
  theme?: 'movie' | 'tv'
}

type ActionItem = {
  title: string
  icon: React.ElementType
  action: 'restart' | 'logout'
  variant?: 'warning' | 'destructive'
}

type MenuItem = NavItem | ActionItem

function isActionItem(item: MenuItem): item is ActionItem {
  return 'action' in item
}

type CollapsibleNavGroup = {
  id: string
  title: string
  icon: React.ElementType
  items: MenuItem[]
}

const libraryNavItems: NavItem[] = [
  { title: 'Movies', href: '/movies', icon: Film, theme: 'movie' },
  { title: 'Series', href: '/series', icon: Tv, theme: 'tv' },
]

const discoverNavItems: NavItem[] = [
  { title: 'Calendar', href: '/calendar', icon: Calendar },
  { title: 'Missing', href: '/missing', icon: Binoculars },
  { title: 'Manual Import', href: '/import', icon: FileInput },
  { title: 'History', href: '/activity/history', icon: History },
]

const settingsGroup: CollapsibleNavGroup = {
  id: 'settings',
  title: 'Settings',
  icon: Settings,
  items: [
    { title: 'Media', href: '/settings/media', icon: FolderOpen },
    { title: 'Downloads', href: '/settings/downloads', icon: Download },
    { title: 'Notifications', href: '/settings/notifications', icon: Bell },
    { title: 'Requests', href: '/settings/requests', icon: Users },
    { title: 'System', href: '/settings/system', icon: Cog },
  ],
}

const systemGroup: CollapsibleNavGroup = {
  id: 'system',
  title: 'System',
  icon: Server,
  items: [
    { title: 'Health', href: '/system/health', icon: HeartPulse },
    { title: 'Scheduled Tasks', href: '/system/tasks', icon: Clock },
    { title: 'Update', href: '/system/update', icon: ArrowUpCircle },
    { title: 'Logs', href: '/system/logs', icon: ScrollText },
    { title: 'Logout', icon: LogOut, action: 'logout', variant: 'warning' },
    { title: 'Restart', icon: RotateCcw, action: 'restart', variant: 'destructive' },
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
  const isActive =
    router.location.pathname === item.href || router.location.pathname.startsWith(`${item.href}/`)

  const iconClassName = cn(
    'size-4 shrink-0 transition-colors',
    item.theme === 'movie' && 'text-movie-500',
    item.theme === 'tv' && 'text-tv-500',
    isActive && item.theme === 'movie' && 'text-movie-400',
    isActive && item.theme === 'tv' && 'text-tv-400',
  )

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

  const linkClassName = cn(
    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all border-l-2 border-transparent',
    collapsed && 'justify-center px-2 border-l-0',
    indented && !collapsed && 'ml-4 border-l border-border pl-4',
    // Default (no theme)
    !item.theme && 'hover:bg-accent hover:text-accent-foreground',
    !item.theme && isActive && 'bg-accent text-accent-foreground',
    // Movie theme
    item.theme === 'movie' && 'hover:bg-movie-500/10 hover:text-foreground',
    item.theme === 'movie' && isActive && 'bg-movie-500/15 text-foreground border-l-movie-500',
    // TV theme
    item.theme === 'tv' && 'hover:bg-tv-500/10 hover:text-foreground',
    item.theme === 'tv' && isActive && 'bg-tv-500/15 text-foreground border-l-tv-500',
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

function MissingBadge() {
  const { data: counts } = useMissingCounts()

  if (!counts || (counts.movies === 0 && counts.episodes === 0)) {
    return null
  }

  return (
    <span className="flex items-center text-xs">
      {counts.movies > 0 && <span className="text-movie-500">{counts.movies}</span>}
      {counts.movies > 0 && counts.episodes > 0 && (
        <span className="text-muted-foreground px-1">|</span>
      )}
      {counts.episodes > 0 && <span className="text-tv-500">{counts.episodes}</span>}
    </span>
  )
}

function NavSection({
  items,
  collapsed,
  includeDownloads = false,
}: {
  items: NavItem[]
  collapsed: boolean
  includeDownloads?: boolean
}) {
  return (
    <div className="space-y-1">
      {items.map((item) => (
        <div key={item.href}>
          {includeDownloads && item.href === '/import' ? (
            <DownloadsNavLink collapsed={collapsed} />
          ) : null}
          <NavLink
            item={item}
            collapsed={collapsed}
            badge={item.href === '/missing' ? <MissingBadge /> : undefined}
          />
        </div>
      ))}
    </div>
  )
}

function CollapsibleNavSection({
  group,
  collapsed: sidebarCollapsed,
  onAction,
}: {
  group: CollapsibleNavGroup
  collapsed: boolean
  onAction?: (action: string) => void
}) {
  const router = useRouterState()
  const { expandedMenus, toggleMenu } = useUIStore()

  const isExpanded = expandedMenus[group.id] ?? false
  const isAnyChildActive = group.items.some(
    (item) => !isActionItem(item) && router.location.pathname === item.href,
  )

  // When sidebar is collapsed, show a popover with submenu items on click
  if (sidebarCollapsed) {
    return (
      <Popover>
        <PopoverTrigger
          className={cn(
            'flex w-full items-center justify-center rounded-md px-2 py-2 text-sm font-medium transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            isAnyChildActive && 'bg-accent text-accent-foreground',
          )}
        >
          <group.icon className="size-5 shrink-0" />
        </PopoverTrigger>
        <PopoverContent side="right" sideOffset={8} className="w-auto min-w-[160px] p-1">
          <div className="text-muted-foreground mb-1 px-2 py-1 text-xs font-semibold">
            {group.title}
          </div>
          {group.items.map((item) =>
            isActionItem(item) ? (
              <button
                key={item.title}
                onClick={() => onAction?.(item.action)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
                  item.variant === 'destructive' &&
                    'text-destructive hover:bg-destructive/10 hover:text-destructive',
                  item.variant === 'warning' &&
                    'text-amber-500 hover:bg-amber-500/10 hover:text-amber-500',
                  !item.variant && 'hover:bg-accent hover:text-accent-foreground',
                )}
              >
                <item.icon className="size-4" />
                <span className="flex-1 text-left">{item.title}</span>
              </button>
            ) : (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  'flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
                  'hover:bg-accent hover:text-accent-foreground',
                  router.location.pathname === item.href &&
                    'bg-accent text-accent-foreground font-medium',
                )}
              >
                <item.icon className="size-4" />
                <span className="flex-1">{item.title}</span>
              </Link>
            ),
          )}
        </PopoverContent>
      </Popover>
    )
  }

  return (
    <Collapsible open={isExpanded} onOpenChange={() => toggleMenu(group.id)}>
      <CollapsibleTrigger
        className={cn(
          'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
          'hover:bg-accent hover:text-accent-foreground',
          isAnyChildActive && 'text-accent-foreground',
        )}
      >
        <group.icon className="size-5 shrink-0" />
        <span className="flex-1 text-left">{group.title}</span>
        <ChevronDown
          className={cn(
            'size-4 shrink-0 transition-transform duration-200',
            isExpanded && 'rotate-180',
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="data-[ending-style]:animate-collapse-up data-[starting-style]:animate-collapse-down overflow-hidden">
        <div className="mt-1 space-y-1">
          {group.items.map((item) =>
            isActionItem(item) ? (
              <button
                key={item.title}
                onClick={() => onAction?.(item.action)}
                className={cn(
                  'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  'border-border ml-4 border-l pl-4',
                  item.variant === 'destructive' &&
                    'text-destructive hover:bg-destructive/10 hover:text-destructive',
                  item.variant === 'warning' &&
                    'text-amber-500 hover:bg-amber-500/10 hover:text-amber-500',
                  !item.variant && 'hover:bg-accent hover:text-accent-foreground',
                )}
              >
                <item.icon className="size-4 shrink-0" />
                <span className="flex-1 text-left">{item.title}</span>
              </button>
            ) : (
              <NavLink key={item.href} item={item} collapsed={false} indented />
            ),
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function Sidebar() {
  const navigate = useNavigate()
  const { sidebarCollapsed, toggleSidebar } = useUIStore()
  const { logout } = usePortalAuthStore()
  const [showRestartDialog, setShowRestartDialog] = useState(false)
  const [showLogoutDialog, setShowLogoutDialog] = useState(false)
  const [countdown, setCountdown] = useState<number | null>(null)
  const restartMutation = useRestart()

  useEffect(() => {
    if (countdown === null) {
      return
    }
    if (countdown === 0) {
      globalThis.location.reload()
      return
    }
    const timer = setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(timer)
  }, [countdown])

  const handleAction = (action: string) => {
    if (action === 'restart') {
      setShowRestartDialog(true)
    } else if (action === 'logout') {
      setShowLogoutDialog(true)
    }
  }

  const handleRestart = async () => {
    await restartMutation.mutateAsync()
    setCountdown(5)
  }

  const handleLogout = () => {
    logout()
    navigate({ to: '/requests/auth/login' })
  }

  return (
    <TooltipProvider delay={0}>
      <aside
        className={cn(
          'border-border bg-card flex h-screen flex-col border-r transition-all duration-300',
          sidebarCollapsed ? 'w-16' : 'w-64',
        )}
      >
        {/* Logo */}
        <div
          className={cn(
            'border-border flex h-14 items-center border-b px-4',
            sidebarCollapsed && 'justify-center px-2',
          )}
        >
          <Link to="/" className="flex items-center gap-2">
            <div className="bg-media-gradient glow-media-sm flex size-8 items-center justify-center rounded-md text-white">
              <Film className="size-5" />
            </div>
            {!sidebarCollapsed && (
              <span className="text-media-gradient text-lg font-semibold">SlipStream</span>
            )}
          </Link>
        </div>

        {/* Navigation */}
        <ScrollArea className="flex-1">
          <nav className="space-y-4 px-3 py-4">
            {/* Library navigation */}
            <NavSection items={libraryNavItems} collapsed={sidebarCollapsed} />

            {/* Divider */}
            <div className="bg-border h-px" />

            {/* Discover navigation */}
            <NavSection items={discoverNavItems} collapsed={sidebarCollapsed} includeDownloads />

            {/* Divider */}
            <div className="bg-border h-px" />

            {/* Settings collapsible menu */}
            <CollapsibleNavSection group={settingsGroup} collapsed={sidebarCollapsed} />

            {/* System collapsible menu */}
            <CollapsibleNavSection
              group={systemGroup}
              collapsed={sidebarCollapsed}
              onAction={handleAction}
            />
          </nav>
        </ScrollArea>

        {/* Collapse toggle */}
        <div className="border-border border-t p-3">
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

      <Dialog
        open={showRestartDialog}
        onOpenChange={(open) => {
          if (countdown === null) {
            setShowRestartDialog(open)
          }
        }}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Confirm Restart</DialogTitle>
            <DialogDescription>
              {countdown === null
                ? 'Are you sure you want to restart the server? The application will be briefly unavailable.'
                : 'Server is restarting. Page will refresh automatically.'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowRestartDialog(false)}
              disabled={countdown !== null}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleRestart}
              disabled={restartMutation.isPending || countdown !== null}
            >
              {countdown === null ? (
                restartMutation.isPending ? (
                  <>
                    <Loader2 className="size-4 animate-spin" />
                    Restarting...
                  </>
                ) : (
                  'Restart'
                )
              ) : (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Restarting ({countdown}s)
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Confirm Logout</DialogTitle>
            <DialogDescription>Are you sure you want to log out?</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowLogoutDialog(false)}>
              Cancel
            </Button>
            <Button className="bg-amber-500 text-white hover:bg-amber-600" onClick={handleLogout}>
              Logout
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  )
}
