import { Download, Globe, Rss, Zap } from 'lucide-react'

import { useDownloadClients, useIndexers } from '@/hooks'

import type { SectionEntry } from '../settings-section-screen'
import { SettingsSectionScreen } from '../settings-section-screen'
import { useHealthWarnings } from '../use-health-warnings'

export function DownloadPipelinePage() {
  const { data: indexers } = useIndexers()
  const { data: clients } = useDownloadClients()
  const warningsFor = useHealthWarnings()

  const entries: SectionEntry[] = [
    {
      title: 'Indexers',
      href: '/settings/download-pipeline/indexers',
      icon: Globe,
      tile: 'bg-tv-600',
      count: indexers?.length,
      warnings: warningsFor(['indexers', 'prowlarr']),
    },
    {
      title: 'Download Clients',
      href: '/settings/download-pipeline/clients',
      icon: Download,
      tile: 'bg-emerald-600',
      count: clients?.length,
      warnings: warningsFor(['downloadClients']),
    },
    {
      title: 'Auto Search',
      href: '/settings/download-pipeline/auto-search',
      icon: Zap,
      tile: 'bg-amber-600',
    },
    {
      title: 'RSS Sync',
      href: '/settings/download-pipeline/rss-sync',
      icon: Rss,
      tile: 'bg-orange-600',
    },
  ]

  return <SettingsSectionScreen title="Download Pipeline" entries={entries} />
}
