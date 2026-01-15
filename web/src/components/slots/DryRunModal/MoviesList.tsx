import { useState } from 'react'
import { ChevronDown, AlertTriangle, HelpCircle, CheckCircle } from 'lucide-react'
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
import type { MoviesListProps, MovieItemProps } from './types'

export function MoviesList({ movies, selectedFileIds, ignoredFileIds, onToggleFileSelection }: MoviesListProps) {
  if (movies.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No movies with files to migrate
      </div>
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

function MovieItem({ movie, selectedFileIds, ignoredFileIds, onToggleFileSelection }: MovieItemProps) {
  const [isOpen, setIsOpen] = useState(movie.hasConflict)
  const problemFiles = movie.files.filter(f => f.conflict || f.needsReview)
  const hasConflictFiles = movie.files.some(f => f.conflict)
  const hasNoMatchFiles = movie.files.some(f => f.needsReview && !f.conflict)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="flex items-center justify-between w-full p-3 border rounded-lg hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-3">
          {(hasConflictFiles || hasNoMatchFiles) ? (
            <Tooltip>
              <TooltipTrigger onClick={(e) => e.stopPropagation()}>
                {hasConflictFiles ? (
                  <AlertTriangle className="size-4 text-orange-500 cursor-help" />
                ) : (
                  <HelpCircle className="size-4 text-red-500 cursor-help" />
                )}
              </TooltipTrigger>
              <TooltipContent side="right" className="max-w-sm">
                <AggregatedFileTooltip files={problemFiles} />
              </TooltipContent>
            </Tooltip>
          ) : (
            <CheckCircle className="size-4 text-green-500" />
          )}
          <div className="text-left">
            <div className="font-medium">
              {movie.title}
              {movie.year && <span className="text-muted-foreground ml-1">({movie.year})</span>}
            </div>
            <div className="text-sm text-muted-foreground">
              {movie.files.length} file{movie.files.length !== 1 ? 's' : ''}
            </div>
          </div>
        </div>
        <ChevronDown className={`size-4 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="mt-1 ml-4 border-l-2 border-muted pl-4 space-y-2 py-2">
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
