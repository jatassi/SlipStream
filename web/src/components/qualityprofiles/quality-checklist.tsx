import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import type { QualityItem } from '@/types'

import { RESOLUTIONS } from './constants'

type QualityChecklistProps = {
  items: QualityItem[]
  onToggle: (qualityId: number) => void
}

type QualityGroupProps = {
  title: string
  description?: string
  items: QualityItem[]
  onToggle: (qualityId: number) => void
}

function resolutionLabel(resolution: number): string {
  if (resolution === 480) {
    return 'SD'
  }
  return `${resolution}p`
}

function QualityGroup({ title, description, items, onToggle }: QualityGroupProps) {
  return (
    <div className="p-3">
      <div className="text-muted-foreground mb-2 text-xs font-medium">{title}</div>
      {description ? <p className="text-muted-foreground mb-2 text-xs">{description}</p> : null}
      <div className="flex flex-wrap gap-x-4 gap-y-1.5">
        {items.map((item) => (
          <label key={item.quality.id} className="flex cursor-pointer items-center gap-2">
            <Checkbox checked={item.allowed} onCheckedChange={() => onToggle(item.quality.id)} />
            <span className="text-sm">{item.quality.name}</span>
          </label>
        ))}
      </div>
    </div>
  )
}

export function QualityChecklist({ items, onToggle }: QualityChecklistProps) {
  const cinemaSourceItems = items.filter((item) => item.quality.resolution === 0)

  return (
    <div className="space-y-2">
      <Label>Allowed Qualities</Label>
      <div className="bg-muted/30 divide-y rounded-lg border">
        {RESOLUTIONS.map((resolution) => {
          const resolutionItems = items.filter((item) => item.quality.resolution === resolution)
          if (resolutionItems.length === 0) {
            return null
          }
          return (
            <QualityGroup
              key={resolution}
              title={resolutionLabel(resolution)}
              items={resolutionItems}
              onToggle={onToggle}
            />
          )
        })}
        {cinemaSourceItems.length > 0 && (
          <QualityGroup
            title="Cinema Source"
            description="Camera recordings of theatrical releases (CAM, HDCAM, TS, HDTS, TELESYNC). Generally very low quality — leave disabled unless you specifically want pre-release copies."
            items={cinemaSourceItems}
            onToggle={onToggle}
          />
        )}
      </div>
    </div>
  )
}
