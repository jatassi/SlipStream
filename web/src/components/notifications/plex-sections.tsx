import { Loader2 } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import type { PlexSection } from './notification-dialog-types'

type PlexSectionsProps = {
  isPlex: boolean
  hasPlexToken: boolean
  serverId: unknown
  sectionIds: number[]
  isLoadingSections: boolean
  plexSections: PlexSection[]
  onSettingChange: (name: string, value: unknown) => void
}

export function PlexSections({
  isPlex,
  hasPlexToken,
  serverId,
  sectionIds,
  isLoadingSections,
  plexSections,
  onSettingChange,
}: PlexSectionsProps) {
  if (!isPlex || !hasPlexToken || !serverId) {return null}

  return (
    <div className="space-y-2">
      <Label>Library Sections</Label>
      <p className="text-muted-foreground text-xs">Select which library sections to refresh</p>
      <PlexSectionsContent
        isLoadingSections={isLoadingSections}
        plexSections={plexSections}
        sectionIds={sectionIds}
        onSettingChange={onSettingChange}
      />
    </div>
  )
}

type PlexSectionsContentProps = {
  isLoadingSections: boolean
  plexSections: PlexSection[]
  sectionIds: number[]
  onSettingChange: (name: string, value: unknown) => void
}

function PlexSectionsContent({
  isLoadingSections,
  plexSections,
  sectionIds,
  onSettingChange,
}: PlexSectionsContentProps) {
  if (isLoadingSections) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" />
        Loading sections...
      </div>
    )
  }

  if (plexSections.length === 0) {
    return <p className="text-muted-foreground text-sm">No movie or TV sections found</p>
  }

  return (
    <div className="space-y-2 rounded-lg border p-3">
      {plexSections.map((section) => (
        <div key={section.key} className="flex items-center justify-between">
          <Label htmlFor={`section-${section.key}`} className="cursor-pointer font-normal">
            {section.title} ({section.type})
          </Label>
          <Switch
            id={`section-${section.key}`}
            checked={sectionIds.includes(section.key)}
            onCheckedChange={(checked) => {
              const newIds = checked
                ? [...sectionIds, section.key]
                : sectionIds.filter((id) => id !== section.key)
              onSettingChange('sectionIds', newIds)
            }}
          />
        </div>
      ))}
    </div>
  )
}
