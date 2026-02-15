import { Link } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight, Film } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useMissingCounts } from '@/hooks'
import { cn } from '@/lib/utils'

import { DownloadsNavLink } from './downloads-nav-link'
import { CollapsibleNavSection } from './sidebar-collapsible-nav'
import { LogoutDialog, RestartDialog } from './sidebar-dialogs'
import { discoverNavItems, libraryNavItems, settingsGroup, systemGroup } from './sidebar-nav-config'
import { NavLink } from './sidebar-nav-link'
import type { NavItem } from './sidebar-types'
import { useSidebarActions } from './use-sidebar'

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

function SidebarLogo({ collapsed }: { collapsed: boolean }) {
  return (
    <div
      className={cn(
        'border-border flex h-14 items-center border-b px-4',
        collapsed && 'justify-center px-2',
      )}
    >
      <Link to="/" className="flex items-center gap-2">
        <div className="bg-media-gradient glow-media-sm flex size-8 items-center justify-center rounded-md text-white">
          <Film className="size-5" />
        </div>
        {!collapsed && (
          <span className="text-media-gradient text-lg font-semibold">SlipStream</span>
        )}
      </Link>
    </div>
  )
}

function CollapseToggle({ collapsed, onToggle }: { collapsed: boolean; onToggle: () => void }) {
  return (
    <div className="border-border border-t p-3">
      <Button
        variant="ghost"
        size="sm"
        onClick={onToggle}
        className={cn('w-full', collapsed && 'px-2')}
      >
        {collapsed ? (
          <ChevronRight className="size-4" />
        ) : (
          <>
            <ChevronLeft className="size-4" />
            <span className="ml-2">Collapse</span>
          </>
        )}
      </Button>
    </div>
  )
}

export function Sidebar() {
  const sidebar = useSidebarActions()

  return (
    <TooltipProvider delay={0}>
      <aside
        className={cn(
          'border-border bg-card flex h-screen flex-col border-r transition-all duration-300',
          sidebar.sidebarCollapsed ? 'w-16' : 'w-64',
        )}
      >
        <SidebarLogo collapsed={sidebar.sidebarCollapsed} />

        <ScrollArea className="flex-1">
          <nav className="space-y-4 px-3 py-4">
            <NavSection items={libraryNavItems} collapsed={sidebar.sidebarCollapsed} />
            <div className="bg-border h-px" />
            <NavSection items={discoverNavItems} collapsed={sidebar.sidebarCollapsed} includeDownloads />
            <div className="bg-border h-px" />
            <CollapsibleNavSection group={settingsGroup} collapsed={sidebar.sidebarCollapsed} />
            <CollapsibleNavSection
              group={systemGroup}
              collapsed={sidebar.sidebarCollapsed}
              onAction={sidebar.handleAction}
            />
          </nav>
        </ScrollArea>

        <CollapseToggle collapsed={sidebar.sidebarCollapsed} onToggle={sidebar.toggleSidebar} />
      </aside>

      <RestartDialog
        open={sidebar.showRestartDialog}
        onOpenChange={sidebar.setShowRestartDialog}
        onRestart={sidebar.handleRestart}
        countdown={sidebar.countdown}
        isPending={sidebar.restartMutation.isPending}
      />

      <LogoutDialog
        open={sidebar.showLogoutDialog}
        onOpenChange={sidebar.setShowLogoutDialog}
        onLogout={sidebar.handleLogout}
      />
    </TooltipProvider>
  )
}
