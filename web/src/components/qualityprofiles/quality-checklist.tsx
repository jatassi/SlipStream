import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import type { QualityItem } from '@/types'

import { RESOLUTIONS } from './constants'

type QualityChecklistProps = {
  items: QualityItem[]
  onToggle: (qualityId: number) => void
}

function resolutionLabel(resolution: number): string {
  if (resolution === 480) {
    return 'SD'
  }
  return `${resolution}p`
}

export function QualityChecklist({ items, onToggle }: QualityChecklistProps) {
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
            <div key={resolution} className="p-3">
              <div className="text-muted-foreground mb-2 text-xs font-medium">
                {resolutionLabel(resolution)}
              </div>
              <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                {resolutionItems.map((item) => (
                  <label key={item.quality.id} className="flex cursor-pointer items-center gap-2">
                    <Checkbox
                      checked={item.allowed}
                      onCheckedChange={() => onToggle(item.quality.id)}
                    />
                    <span className="text-sm">{item.quality.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
