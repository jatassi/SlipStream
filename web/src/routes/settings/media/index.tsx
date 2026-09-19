import { ArrowRightLeft, FileInput, FolderOpen, Layers, Sliders } from 'lucide-react'

import { useQualityProfiles, useRootFolders, useSlots } from '@/hooks'

import type { SectionEntry } from '../settings-section-screen'
import { SettingsSectionScreen } from '../settings-section-screen'
import { useHealthWarnings } from '../use-health-warnings'

export function MediaSettingsPage() {
  const { data: rootFolders } = useRootFolders()
  const { data: profiles } = useQualityProfiles()
  const { data: slots } = useSlots()
  const warningsFor = useHealthWarnings()

  const entries: SectionEntry[] = [
    {
      title: 'Root Folders',
      href: '/settings/media/root-folders',
      icon: FolderOpen,
      tile: 'bg-movie-600',
      count: rootFolders?.length,
      warnings: warningsFor(['rootFolders', 'storage']),
    },
    {
      title: 'Quality Profiles',
      href: '/settings/media/quality-profiles',
      icon: Sliders,
      tile: 'bg-emerald-600',
      count: profiles?.length,
    },
    {
      title: 'Version Slots',
      href: '/settings/media/version-slots',
      icon: Layers,
      tile: 'bg-violet-600',
      count: slots?.filter((slot) => slot.enabled).length,
    },
    {
      title: 'Import & Naming',
      href: '/settings/media/file-naming',
      icon: FileInput,
      tile: 'bg-sky-600',
    },
    {
      title: 'Migrate from *arr',
      href: '/settings/media/arr-import',
      icon: ArrowRightLeft,
      tile: 'bg-zinc-600',
    },
  ]

  return <SettingsSectionScreen title="Media" entries={entries} />
}
