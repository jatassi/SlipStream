import { FileVideo, Layers } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { SlotRejectionInfo } from '@/types'

import type { FileItemProps } from './types'
import { formatBytes, getFileName } from './utils'

function SlotRejectionsList({ rejections }: { rejections: SlotRejectionInfo[] }) {
  return (
    <div className="space-y-2">
      {rejections.map((rejection) => (
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
  )
}

function ConflictBadge({ file, compactClass }: { file: FileItemProps['file']; compactClass: string }) {
  const rejections = file.slotRejections ?? []
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
        {rejections.length > 0 ? <SlotRejectionsList rejections={rejections} /> : null}
      </TooltipContent>
    </Tooltip>
  )
}

function NoMatchBadge({ file, compactClass }: { file: FileItemProps['file']; compactClass: string }) {
  const rejections = file.slotRejections ?? []
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
        {rejections.length > 0 ? <SlotRejectionsList rejections={rejections} /> : null}
      </TooltipContent>
    </Tooltip>
  )
}

function FileBadge({
  file,
  isIgnored,
  isCompact,
}: {
  file: FileItemProps['file']
  isIgnored: boolean
  isCompact: boolean
}) {
  const compactClass = isCompact ? 'text-[10px] px-1.5 py-0 h-4' : ''

  if (isIgnored) {
    return (
      <Badge variant="outline" className={`text-muted-foreground ${compactClass}`}>
        Ignored
      </Badge>
    )
  }
  if (file.conflict) {
    return <ConflictBadge file={file} compactClass={compactClass} />
  }
  if (file.needsReview) {
    return <NoMatchBadge file={file} compactClass={compactClass} />
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

function CompactFileItem({ file, isSelected, isIgnored, onToggleSelection, fileName, fileSize }: {
  file: FileItemProps['file']
  isSelected: boolean
  isIgnored: boolean
  onToggleSelection: () => void
  fileName: string
  fileSize: string
}) {
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
        <FileBadge file={file} isIgnored={isIgnored} isCompact />
      </div>
    </div>
  )
}

function FullFileItem({ file, isSelected, isIgnored, onToggleSelection, fileName, fileSize }: {
  file: FileItemProps['file']
  isSelected: boolean
  isIgnored: boolean
  onToggleSelection: () => void
  fileName: string
  fileSize: string
}) {
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
      <div className="ml-3 flex shrink-0 items-center gap-2">
        <FileBadge file={file} isIgnored={isIgnored} isCompact={false} />
      </div>
    </div>
  )
}

export function FileItem({
  file,
  compact = false,
  isSelected,
  isIgnored = false,
  onToggleSelection,
}: FileItemProps) {
  const fileName = getFileName(file.path)
  const fileSize = formatBytes(file.size)

  if (compact) {
    return (
      <CompactFileItem
        file={file}
        isSelected={isSelected}
        isIgnored={isIgnored}
        onToggleSelection={onToggleSelection}
        fileName={fileName}
        fileSize={fileSize}
      />
    )
  }

  return (
    <FullFileItem
      file={file}
      isSelected={isSelected}
      isIgnored={isIgnored}
      onToggleSelection={onToggleSelection}
      fileName={fileName}
      fileSize={fileSize}
    />
  )
}
