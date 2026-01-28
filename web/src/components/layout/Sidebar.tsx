import { useState, useEffect } from 'react'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import {
  Film,
  Tv,
  Calendar,
  Activity,
  History,
  Settings,
  FolderOpen,
  Download,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Clock,
  Search,
  HeartPulse,
  Server,
  FileInput,
  Bell,
  Cog,
  Users,
  RotateCcw,
  Loader2,
  LogOut,
  ArrowUpCircle,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { useUIStore, useDownloadingStore, usePortalAuthStore } from '@/stores'
import { useMissingCounts, useRestart } from '@/hooks'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'

interface NavItem {
  title: string
  href: string
  icon: React.ElementType
}

interface ActionItem {
  title: string
  icon: React.ElementType
  action: 'restart' | 'logout'
  variant?: 'warning' | 'destructive'
}

type MenuItem = NavItem | ActionItem

function isActionItem(item: MenuItem): item is ActionItem {
  return 'action' in item
}

interface CollapsibleNavGroup {
  id: string
  title: string
  icon: React.ElementType
  items: MenuItem[]
}

const libraryNavItems: NavItem[] = [
  { title: 'Movies', href: '/movies', icon: Film },
  { title: 'Series', href: '/series', icon: Tv },
]

const discoverNavItems: NavItem[] = [
  { title: 'Calendar', href: '/calendar', icon: Calendar },
  { title: 'Missing', href: '/missing', icon: Search },
  { title: 'Manual Import', href: '/import', icon: FileInput },
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

function DownloadsBadge() {
  const queueItems = useDownloadingStore((state) => state.queueItems)
  const count = queueItems.length

  if (count === 0) {
    return null
  }

  return (
    <span className="text-xs text-muted-foreground">
      {count}
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
    (item) => !isActionItem(item) && router.location.pathname === item.href
  )

  // When sidebar is collapsed, show a popover with submenu items on click
  if (sidebarCollapsed) {
    return (
      <Popover>
        <PopoverTrigger
          className={cn(
            'flex w-full items-center justify-center rounded-md px-2 py-2 text-sm font-medium transition-colors',
            'hover:bg-accent hover:text-accent-foreground',
            isAnyChildActive && 'bg-accent text-accent-foreground'
          )}
        >
          <group.icon className="size-5 shrink-0" />
        </PopoverTrigger>
        <PopoverContent
          side="right"
          sideOffset={8}
          className="w-auto min-w-[160px] p-1"
        >
          <div className="mb-1 px-2 py-1 text-xs font-semibold text-muted-foreground">
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
                  !item.variant && 'hover:bg-accent hover:text-accent-foreground'
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
                    'bg-accent text-accent-foreground font-medium'
                )}
              >
                <item.icon className="size-4" />
                <span className="flex-1">{item.title}</span>
                {item.href === '/activity' && <DownloadsBadge />}
              </Link>
            )
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
          {group.items.map((item) =>
            isActionItem(item) ? (
              <button
                key={item.title}
                onClick={() => onAction?.(item.action)}
                className={cn(
                  'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  'ml-4 border-l border-border pl-4',
                  item.variant === 'destructive' &&
                    'text-destructive hover:bg-destructive/10 hover:text-destructive',
                  item.variant === 'warning' &&
                    'text-amber-500 hover:bg-amber-500/10 hover:text-amber-500',
                  !item.variant && 'hover:bg-accent hover:text-accent-foreground'
                )}
              >
                <item.icon className="size-4 shrink-0" />
                <span className="flex-1 text-left">{item.title}</span>
              </button>
            ) : (
              <NavLink
                key={item.href}
                item={item}
                collapsed={false}
                indented
                badge={item.href === '/activity' ? <DownloadsBadge /> : undefined}
              />
            )
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
    if (countdown === null) return
    if (countdown === 0) {
      window.location.reload()
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
            {/* Library navigation */}
            <NavSection items={libraryNavItems} collapsed={sidebarCollapsed} />

            {/* Divider */}
            <div className="h-px bg-border" />

            {/* Discover navigation */}
            <NavSection items={discoverNavItems} collapsed={sidebarCollapsed} />

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
              onAction={handleAction}
            />
          </nav>
        </ScrollArea>

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

      <Dialog
        open={showRestartDialog}
        onOpenChange={(open) => {
          if (countdown === null) setShowRestartDialog(open)
        }}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Confirm Restart</DialogTitle>
            <DialogDescription>
              {countdown !== null
                ? 'Server is restarting. Page will refresh automatically.'
                : 'Are you sure you want to restart the server? The application will be briefly unavailable.'}
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
              {countdown !== null ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Restarting ({countdown}s)
                </>
              ) : restartMutation.isPending ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Restarting...
                </>
              ) : (
                'Restart'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Confirm Logout</DialogTitle>
            <DialogDescription>
              Are you sure you want to log out?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowLogoutDialog(false)}
            >
              Cancel
            </Button>
            <Button
              className="bg-amber-500 text-white hover:bg-amber-600"
              onClick={handleLogout}
            >
              Logout
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  )
}
