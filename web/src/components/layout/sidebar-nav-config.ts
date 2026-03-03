import {
  Binoculars,
  Calendar,
  Cog,
  FileInput,
  Film,
  FolderOpen,
  History,
  LogOut,
  RotateCcw,
  Server,
  Settings,
  Tv,
  Users,
  Workflow,
} from 'lucide-react'

import type { ActionItem, CollapsibleNavGroup, NavItem } from './sidebar-types'

export const libraryNavItems: NavItem[] = [
  { title: 'Movies', href: '/movies', icon: Film, theme: 'movie' },
  { title: 'Series', href: '/series', icon: Tv, theme: 'tv' },
]

export const discoverNavItems: NavItem[] = [
  { title: 'Calendar', href: '/calendar', icon: Calendar },
  { title: 'Requests', href: '/requests-admin/queue', icon: Users, activePrefix: '/requests-admin' },
  { title: 'Missing', href: '/missing', icon: Binoculars },
  { title: 'Manual Import', href: '/import', icon: FileInput },
  { title: 'History', href: '/history', icon: History },
]

export const settingsGroup: CollapsibleNavGroup = {
  id: 'settings',
  title: 'Settings',
  icon: Settings,
  items: [
    { title: 'Media', href: '/settings/media', icon: FolderOpen },
    { title: 'Download Pipeline', href: '/settings/download-pipeline', icon: Workflow },
    { title: 'General', href: '/settings/general', icon: Cog },
  ],
}

export const systemNavItem: NavItem = {
  title: 'System',
  href: '/system/health',
  icon: Server,
  activePrefix: '/system',
}

export const standaloneActions: ActionItem[] = [
  { title: 'Logout', icon: LogOut, action: 'logout', variant: 'warning' },
  { title: 'Restart', icon: RotateCcw, action: 'restart', variant: 'destructive' },
]
