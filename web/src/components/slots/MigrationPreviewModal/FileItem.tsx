import { FileVideo, Layers } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import type { FileItemProps } from './types'
import { formatBytes, getFileName } from './utils'

export function FileItem({
  file,
  compact = false,
  isSelected,
  isIgnored = false,
  onToggleSelection,
}: FileItemProps) {
  const fileName = getFileName(file.path)
  const fileSize = formatBytes(file.size)

  const renderBadge = (isCompact: boolean) => {
    const compactClass = isCompact ? 'text-[10px] px-1.5 py-0 h-4' : ''

    if (isIgnored) {
      return (
        <Badge variant="outline" className={`text-muted-foreground ${compactClass}`}>
          Ignored
        </Badge>
      )
    }
    if (file.conflict) {
      return (
        <Tooltip>
          <TooltipTrigger>
            <Badge
              variant="outline"
              className={`cursor-help bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400 ${compactClass}`}
            >
              Conflict
            </Badge>
          </TooltipTrigger>
          <TooltipContent side="left" className="max-w-sm">
            <p className="mb-2 font-medium">{file.conflict}</p>
            {file.slotRejections && file.slotRejections.length > 0 ? (
              <div className="space-y-2">
                {file.slotRejections.map((rejection) => (
                  <div key={rejection.slotId} className="text-xs">
                    <span className="text-muted-foreground font-medium">{rejection.slotName}:</span>
                    <ul className="mt-0.5 ml-1 list-inside list-disc">
                      {rejection.reasons.map((reason) => (
                        <li key={reason}>{reason}</li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            ) : null}
          </TooltipContent>
        </Tooltip>
      )
    }
    if (file.needsReview) {
      return (
        <Tooltip>
          <TooltipTrigger>
            <Badge
              variant="outline"
              className={`cursor-help bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 ${compactClass}`}
            >
              No Match
            </Badge>
          </TooltipTrigger>
          <TooltipContent side="left" className="max-w-sm">
            {file.slotRejections && file.slotRejections.length > 0 ? (
              <div className="space-y-2">
                {file.slotRejections.map((rejection) => (
                  <div key={rejection.slotId} className="text-xs">
                    <span className="text-muted-foreground font-medium">{rejection.slotName}:</span>
                    <ul className="mt-0.5 ml-1 list-inside list-disc">
                      {rejection.reasons.map((reason) => (
                        <li key={reason}>{reason}</li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            ) : null}
          </TooltipContent>
        </Tooltip>
      )
    }
    if (file.proposedSlotName) {
      return (
        <Badge
          variant="outline"
          className={`bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 ${compactClass}`}
        >
          <Layers className={isCompact ? 'mr-1 size-2.5' : 'mr-1 size-3'} />
          {file.proposedSlotName}
        </Badge>
      )
    }
    return (
      <Badge variant="outline" className={compactClass}>
        No Slot
      </Badge>
    )
  }

  if (compact) {
    return (
      <div
        className={`flex items-center justify-between rounded px-2 py-1 text-xs ${isSelected ? 'bg-primary/10 ring-primary/30 ring-1' : 'bg-muted/30'}`}
      >
        <div className="flex min-w-0 items-center gap-2">
          <Checkbox checked={isSelected} onCheckedChange={onToggleSelection} className="size-3" />
          <FileVideo className="text-muted-foreground size-3 shrink-0" />
          <Tooltip>
            <TooltipTrigger
              render={
                <span
                  className={`cursor-help truncate ${isIgnored ? 'text-muted-foreground line-through' : ''}`}
                />
              }
            >
              {fileName}
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-md">
              <p className="break-all">{file.path}</p>
            </TooltipContent>
          </Tooltip>
        </div>
        <div className="ml-2 flex shrink-0 items-center gap-2">
          <span className="text-muted-foreground">{fileSize}</span>
          {renderBadge(true)}
        </div>
      </div>
    )
  }

  return (
    <div
      className={`flex items-center justify-between rounded px-3 py-2 text-sm ${isSelected ? 'bg-primary/10 ring-primary/30 ring-1' : 'bg-muted/30'}`}
    >
      <div className="flex min-w-0 items-center gap-3">
        <Checkbox checked={isSelected} onCheckedChange={onToggleSelection} className="size-4" />
        <FileVideo className="text-muted-foreground size-4 shrink-0" />
        <div className="min-w-0">
          <Tooltip>
            <TooltipTrigger
              render={
                <div
                  className={`cursor-help truncate font-medium ${isIgnored ? 'text-muted-foreground font-normal line-through' : ''}`}
                />
              }
            >
              {fileName}
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-md">
              <p className="break-all">{file.path}</p>
            </TooltipContent>
          </Tooltip>
          <div className="text-muted-foreground flex items-center gap-2 text-xs">
            <span>{file.quality}</span>
            <span>•</span>
            <span>{fileSize}</span>
            {file.matchScore > 0 && (
              <>
                <span>•</span>
                <span>Score: {file.matchScore.toFixed(1)}</span>
              </>
            )}
          </div>
        </div>
      </div>
      <div className="ml-3 flex shrink-0 items-center gap-2">{renderBadge(false)}</div>
    </div>
  )
}
