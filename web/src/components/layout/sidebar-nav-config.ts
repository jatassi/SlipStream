import {
  ArrowUpCircle,
  Bell,
  Binoculars,
  Calendar,
  Clock,
  Cog,
  Download,
  FileInput,
  Film,
  FolderOpen,
  HeartPulse,
  History,
  LogOut,
  RotateCcw,
  ScrollText,
  Server,
  Settings,
  Tv,
  Users,
} from 'lucide-react'

import type { CollapsibleNavGroup, NavItem } from './sidebar-types'

export const libraryNavItems: NavItem[] = [
  { title: 'Movies', href: '/movies', icon: Film, theme: 'movie' },
  { title: 'Series', href: '/series', icon: Tv, theme: 'tv' },
]

export const discoverNavItems: NavItem[] = [
  { title: 'Calendar', href: '/calendar', icon: Calendar },
  { title: 'Missing', href: '/missing', icon: Binoculars },
  { title: 'Manual Import', href: '/import', icon: FileInput },
  { title: 'History', href: '/activity/history', icon: History },
]

export const settingsGroup: CollapsibleNavGroup = {
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

export const systemGroup: CollapsibleNavGroup = {
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
