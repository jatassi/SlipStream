import { useState } from 'react'

import { AlertTriangle, CheckCircle, ChevronDown, HelpCircle, XCircle } from 'lucide-react'

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { FileMigrationPreview } from '@/types'

import { AggregatedFileTooltip } from './aggregated-file-tooltip'
import { FileItem } from './file-item'
import type { EpisodeItemProps, SeasonItemProps, TVShowItemProps, TVShowsListProps } from './types'

function analyzeFileIssues(files: FileMigrationPreview[]) {
  const hasConflict = files.some((f) => f.conflict)
  const hasNoMatch = files.some((f) => f.needsReview && !f.conflict)
  return { hasConflict, hasNoMatch, hasMixed: hasConflict && hasNoMatch }
}

function FileStatusIcon({ hasConflict, hasNoMatch, size }: {
  hasConflict: boolean
  hasNoMatch: boolean
  size: 'sm' | 'md'
}) {
  const cls = size === 'md' ? 'size-4' : 'size-3'
  const hasMixed = hasConflict && hasNoMatch

  if (hasMixed) {
    return <XCircle className={`${cls} text-red-500`} />
  }
  if (hasConflict) {
    return <AlertTriangle className={`${cls} text-orange-500`} />
  }
  if (hasNoMatch) {
    return <HelpCircle className={`${cls} text-red-500`} />
  }
  return size === 'md' ? <CheckCircle className={`${cls} text-green-500`} /> : null
}

function EpisodeStatusIndicator({ episode }: { episode: EpisodeItemProps['episode'] }) {
  const problemFiles = episode.files.filter((f) => f.conflict ?? f.needsReview)
  const hasConflict = episode.files.some((f) => f.conflict)
  const hasNoMatch = episode.files.some((f) => f.needsReview && !f.conflict)

  if (!hasConflict && !hasNoMatch) {
    return <CheckCircle className="size-3 text-green-500" />
  }

  return (
    <Tooltip>
      <TooltipTrigger onClick={(e) => e.stopPropagation()}>
        {hasConflict ? (
          <AlertTriangle className="size-3 cursor-help text-orange-500" />
        ) : (
          <HelpCircle className="size-3 cursor-help text-red-500" />
        )}
      </TooltipTrigger>
      <TooltipContent side="right" className="max-w-sm">
        <AggregatedFileTooltip files={problemFiles} />
      </TooltipContent>
    </Tooltip>
  )
}

export function TVShowsList({
  shows,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: TVShowsListProps) {
  if (shows.length === 0) {
    return (
      <div className="text-muted-foreground py-8 text-center">No series with files to migrate</div>
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

function TVShowItem({
  show,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: TVShowItemProps) {
  const [isOpen, setIsOpen] = useState(show.hasConflict)
  const allFiles = show.seasons.flatMap((s) => s.episodes.flatMap((e) => e.files))
  const { hasConflict, hasNoMatch } = analyzeFileIssues(allFiles)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between rounded-lg border p-3 transition-colors">
        <div className="flex items-center gap-3">
          <FileStatusIcon hasConflict={hasConflict} hasNoMatch={hasNoMatch} size="md" />
          <div className="text-left">
            <div className="font-medium">{show.title}</div>
            <div className="text-muted-foreground text-sm">
              {show.seasons.length} season{show.seasons.length === 1 ? '' : 's'} • {show.totalFiles}{' '}
              file{show.totalFiles === 1 ? '' : 's'}
            </div>
          </div>
        </div>
        <ChevronDown
          className={`text-muted-foreground size-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="border-muted mt-1 ml-4 space-y-2 border-l-2 py-2 pl-4">
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

function SeasonItem({
  season,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: SeasonItemProps) {
  const [isOpen, setIsOpen] = useState(season.hasConflict)
  const allFiles = season.episodes.flatMap((e) => e.files)
  const { hasConflict, hasNoMatch } = analyzeFileIssues(allFiles)
  const hasAnyIssue = hasConflict || hasNoMatch

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between rounded p-2 transition-colors">
        <div className="flex items-center gap-2">
          {hasAnyIssue ? <FileStatusIcon hasConflict={hasConflict} hasNoMatch={hasNoMatch} size="sm" /> : null}
          <span className="text-sm font-medium">Season {season.seasonNumber}</span>
          <span className="text-muted-foreground text-xs">
            ({season.episodes.length} episodes, {season.totalFiles} files)
          </span>
        </div>
        <ChevronDown
          className={`text-muted-foreground size-3 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="border-muted ml-4 space-y-1 border-l py-1 pl-3">
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

function EpisodeItem({
  episode,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: EpisodeItemProps) {
  const [isOpen, setIsOpen] = useState(episode.hasConflict)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="hover:bg-muted/30 flex w-full items-center justify-between rounded p-1.5 text-sm transition-colors">
        <div className="flex items-center gap-2">
          <EpisodeStatusIndicator episode={episode} />
          <span>E{String(episode.episodeNumber).padStart(2, '0')}</span>
          {episode.title ? (
            <span className="text-muted-foreground max-w-[200px] truncate">{episode.title}</span>
          ) : null}
          <span className="text-muted-foreground text-xs">
            ({episode.files.length} file{episode.files.length === 1 ? '' : 's'})
          </span>
        </div>
        <ChevronDown
          className={`text-muted-foreground size-3 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="ml-4 space-y-1 py-1 pl-2">
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
