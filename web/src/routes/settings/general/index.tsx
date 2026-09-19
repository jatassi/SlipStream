import { Bell, Lock, Server } from 'lucide-react'

import { useNotifications } from '@/hooks'

import type { SectionEntry } from '../settings-section-screen'
import { SettingsSectionScreen } from '../settings-section-screen'

export function GeneralSettingsPage() {
  const { data: notifications } = useNotifications()

  const entries: SectionEntry[] = [
    {
      title: 'Server',
      href: '/settings/general/server',
      icon: Server,
      tile: 'bg-zinc-600',
    },
    {
      title: 'Authentication',
      href: '/settings/general/authentication',
      icon: Lock,
      tile: 'bg-rose-600',
    },
    {
      title: 'Notifications',
      href: '/settings/general/notifications',
      icon: Bell,
      tile: 'bg-sky-600',
      count: notifications?.length,
    },
  ]

  return <SettingsSectionScreen title="General" entries={entries} />
}
