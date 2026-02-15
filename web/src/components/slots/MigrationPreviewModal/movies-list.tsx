import { useState } from 'react'

import { AlertTriangle, CheckCircle, ChevronDown, HelpCircle } from 'lucide-react'

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { FileMigrationPreview } from '@/types'

import { AggregatedFileTooltip } from './aggregated-file-tooltip'
import { FileItem } from './file-item'
import type { MovieItemProps, MoviesListProps } from './types'

function StatusIcon({
  hasConflictFiles,
  hasNoMatchFiles,
  problemFiles,
}: {
  hasConflictFiles: boolean
  hasNoMatchFiles: boolean
  problemFiles: FileMigrationPreview[]
}) {
  if (!hasConflictFiles && !hasNoMatchFiles) {
    return <CheckCircle className="size-4 text-green-500" />
  }

  return (
    <Tooltip>
      <TooltipTrigger onClick={(e) => e.stopPropagation()}>
        {hasConflictFiles ? (
          <AlertTriangle className="size-4 cursor-help text-orange-500" />
        ) : (
          <HelpCircle className="size-4 cursor-help text-red-500" />
        )}
      </TooltipTrigger>
      <TooltipContent side="right" className="max-w-sm">
        <AggregatedFileTooltip files={problemFiles} />
      </TooltipContent>
    </Tooltip>
  )
}

function MovieTitle({ title, year }: { title: string; year?: number }) {
  return (
    <div className="font-medium">
      {title}
      {year ? <span className="text-muted-foreground ml-1">({year})</span> : null}
    </div>
  )
}

export function MoviesList({
  movies,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: MoviesListProps) {
  if (movies.length === 0) {
    return (
      <div className="text-muted-foreground py-8 text-center">No movies with files to migrate</div>
    )
  }

  return (
    <div className="space-y-2">
      {movies.map((movie) => (
        <MovieItem
          key={movie.movieId}
          movie={movie}
          selectedFileIds={selectedFileIds}
          ignoredFileIds={ignoredFileIds}
          onToggleFileSelection={onToggleFileSelection}
        />
      ))}
    </div>
  )
}

function MovieItem({
  movie,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: MovieItemProps) {
  const [isOpen, setIsOpen] = useState(movie.hasConflict)
  const problemFiles = movie.files.filter((f) => f.conflict ?? f.needsReview)
  const hasConflictFiles = movie.files.some((f) => f.conflict)
  const hasNoMatchFiles = movie.files.some((f) => f.needsReview && !f.conflict)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between rounded-lg border p-3 transition-colors">
        <div className="flex items-center gap-3">
          <StatusIcon
            hasConflictFiles={hasConflictFiles}
            hasNoMatchFiles={hasNoMatchFiles}
            problemFiles={problemFiles}
          />
          <div className="text-left">
            <MovieTitle title={movie.title} year={movie.year} />
            <div className="text-muted-foreground text-sm">
              {movie.files.length} file{movie.files.length === 1 ? '' : 's'}
            </div>
          </div>
        </div>
        <ChevronDown
          className={`text-muted-foreground size-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="border-muted mt-1 ml-4 space-y-2 border-l-2 py-2 pl-4">
          {movie.files.map((file) => (
            <FileItem
              key={file.fileId}
              file={file}
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
