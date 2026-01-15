import { useState } from 'react'
import { ChevronDown, AlertTriangle, HelpCircle, CheckCircle, XCircle } from 'lucide-react'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { AggregatedFileTooltip } from './AggregatedFileTooltip'
import { FileItem } from './FileItem'
import type { TVShowsListProps, TVShowItemProps, SeasonItemProps, EpisodeItemProps } from './types'

export function TVShowsList({ shows, selectedFileIds, ignoredFileIds, onToggleFileSelection }: TVShowsListProps) {
  if (shows.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No series with files to migrate
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {shows.map((show) => (
        <TVShowItem
          key={show.seriesId}
          show={show}
          selectedFileIds={selectedFileIds}
          ignoredFileIds={ignoredFileIds}
          onToggleFileSelection={onToggleFileSelection}
        />
      ))}
    </div>
  )
}

function TVShowItem({ show, selectedFileIds, ignoredFileIds, onToggleFileSelection }: TVShowItemProps) {
  const [isOpen, setIsOpen] = useState(show.hasConflict)

  const allFiles = show.seasons.flatMap(s => s.episodes.flatMap(e => e.files))
  const hasConflictFiles = allFiles.some(f => f.conflict)
  const hasNoMatchFiles = allFiles.some(f => f.needsReview && !f.conflict)
  const hasMixedIssues = hasConflictFiles && hasNoMatchFiles

  const getIcon = () => {
    if (hasMixedIssues) return <XCircle className="size-4 text-red-500" />
    if (hasConflictFiles) return <AlertTriangle className="size-4 text-orange-500" />
    if (hasNoMatchFiles) return <HelpCircle className="size-4 text-red-500" />
    return <CheckCircle className="size-4 text-green-500" />
  }

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-3 border rounded-lg hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-3">
          {getIcon()}
          <div className="text-left">
            <div className="font-medium">{show.title}</div>
            <div className="text-sm text-muted-foreground">
              {show.seasons.length} season{show.seasons.length !== 1 ? 's' : ''} • {show.totalFiles} file{show.totalFiles !== 1 ? 's' : ''}
            </div>
          </div>
        </div>
        <ChevronDown className={`size-4 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="mt-1 ml-4 border-l-2 border-muted pl-4 space-y-2 py-2">
          {show.seasons.map((season) => (
            <SeasonItem
              key={season.seasonNumber}
              season={season}
              selectedFileIds={selectedFileIds}
              ignoredFileIds={ignoredFileIds}
              onToggleFileSelection={onToggleFileSelection}
            />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function SeasonItem({ season, selectedFileIds, ignoredFileIds, onToggleFileSelection }: SeasonItemProps) {
  const [isOpen, setIsOpen] = useState(season.hasConflict)

  const allFiles = season.episodes.flatMap(e => e.files)
  const hasConflictFiles = allFiles.some(f => f.conflict)
  const hasNoMatchFiles = allFiles.some(f => f.needsReview && !f.conflict)
  const hasMixedIssues = hasConflictFiles && hasNoMatchFiles
  const hasAnyIssue = hasConflictFiles || hasNoMatchFiles

  const getIcon = () => {
    if (hasMixedIssues) return <XCircle className="size-3 text-red-500" />
    if (hasConflictFiles) return <AlertTriangle className="size-3 text-orange-500" />
    if (hasNoMatchFiles) return <HelpCircle className="size-3 text-red-500" />
    return null
  }

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-2 rounded hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-2">
          {hasAnyIssue && getIcon()}
          <span className="font-medium text-sm">Season {season.seasonNumber}</span>
          <span className="text-xs text-muted-foreground">
            ({season.episodes.length} episodes, {season.totalFiles} files)
          </span>
        </div>
        <ChevronDown className={`size-3 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="ml-4 border-l border-muted pl-3 space-y-1 py-1">
          {season.episodes.map((episode) => (
            <EpisodeItem
              key={episode.episodeId}
              episode={episode}
              selectedFileIds={selectedFileIds}
              ignoredFileIds={ignoredFileIds}
              onToggleFileSelection={onToggleFileSelection}
            />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function EpisodeItem({ episode, selectedFileIds, ignoredFileIds, onToggleFileSelection }: EpisodeItemProps) {
  const [isOpen, setIsOpen] = useState(episode.hasConflict)
  const problemFiles = episode.files.filter(f => f.conflict || f.needsReview)
  const hasConflictFiles = episode.files.some(f => f.conflict)
  const hasNoMatchFiles = episode.files.some(f => f.needsReview && !f.conflict)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-1.5 rounded hover:bg-muted/30 transition-colors text-sm">
        <div className="flex items-center gap-2">
          {(hasConflictFiles || hasNoMatchFiles) ? (
            <Tooltip>
              <TooltipTrigger onClick={(e) => e.stopPropagation()}>
                {hasConflictFiles ? (
                  <AlertTriangle className="size-3 text-orange-500 cursor-help" />
                ) : (
                  <HelpCircle className="size-3 text-red-500 cursor-help" />
                )}
              </TooltipTrigger>
              <TooltipContent side="right" className="max-w-sm">
                <AggregatedFileTooltip files={problemFiles} />
              </TooltipContent>
            </Tooltip>
          ) : (
            <CheckCircle className="size-3 text-green-500" />
          )}
          <span>E{String(episode.episodeNumber).padStart(2, '0')}</span>
          {episode.title && <span className="text-muted-foreground truncate max-w-[200px]">{episode.title}</span>}
          <span className="text-xs text-muted-foreground">({episode.files.length} file{episode.files.length !== 1 ? 's' : ''})</span>
        </div>
        <ChevronDown className={`size-3 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="ml-4 pl-2 space-y-1 py-1">
          {episode.files.map((file) => (
            <FileItem
              key={file.fileId}
              file={file}
              compact
              isSelected={selectedFileIds.has(file.fileId)}
              isIgnored={ignoredFileIds.has(file.fileId)}
              onToggleSelection={() => onToggleFileSelection(file.fileId)}
            />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
