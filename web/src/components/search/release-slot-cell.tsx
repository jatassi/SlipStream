import { ArrowUp, Layers } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { TorrentInfo } from '@/types'

function getSlotTooltip(release: TorrentInfo): string {
  if (release.isSlotUpgrade) {return 'Will upgrade existing file in this slot'}
  if (release.isSlotNewFill) {return 'Will fill empty slot'}
  return `Target: ${release.targetSlotName}`
}

export function ReleaseSlotCell({ release }: { release: TorrentInfo }) {
  if (!release.targetSlotName) {
    return <span className="text-muted-foreground">-</span>
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <div className="flex items-center gap-1">
            <Layers className="size-3" />
            <span className="text-sm">{release.targetSlotName}</span>
            {release.isSlotUpgrade ? (
              <Badge variant="secondary" className="px-1 text-xs">
                <ArrowUp className="size-3" />
              </Badge>
            ) : null}
            {release.isSlotNewFill ? (
              <Badge
                variant="outline"
                className="border-green-500 px-1 text-xs text-green-500"
              >
                New
              </Badge>
            ) : null}
          </div>
        </TooltipTrigger>
        <TooltipContent>{getSlotTooltip(release)}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
