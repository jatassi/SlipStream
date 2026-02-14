import { useCallback, useEffect, useMemo, useState } from 'react'

import {
  AlertTriangle,
  Ban,
  Bug,
  Check,
  CheckSquare,
  FileVideo,
  Film,
  HelpCircle,
  Layers,
  Loader2,
  RotateCcw,
  Square,
  Tv,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useDeveloperMode, useExecuteMigration, useMigrationPreview, useSlots } from '@/hooks'

import { AssignModal } from './AssignModal'
import { generateDebugPreview } from './debug'
import { MoviesList } from './MoviesList'
import { TVShowsList } from './SeriesList'
import { SummaryCard } from './SummaryCard'
import type {
  DryRunModalProps,
  FileMigrationPreview,
  ManualEdit,
  MigrationPreview,
  MovieMigrationPreview,
  TVShowMigrationPreview,
} from './types'

export function DryRunModal({ open, onOpenChange, onMigrationComplete }: DryRunModalProps) {
  const previewMutation = useMigrationPreview()
  const executeMutation = useExecuteMigration()
  const developerMode = useDeveloperMode()
  const { data: slots = [] } = useSlots()

  const [preview, setPreview] = useState<MigrationPreview | null>(null)
  const [activeTab, setActiveTab] = useState<'movies' | 'tv'>('movies')
  const [isDebugData, setIsDebugData] = useState(false)
  const [isLoadingDebugData, setIsLoadingDebugData] = useState(false)
  const [filter, setFilter] = useState<'all' | 'assigned' | 'conflicts' | 'nomatch'>('all')

  // Selection and manual edit state
  const [selectedFileIds, setSelectedFileIds] = useState<Set<number>>(new Set())
  const [manualEdits, setManualEdits] = useState<Map<number, ManualEdit>>(new Map())
  const [assignModalOpen, setAssignModalOpen] = useState(false)

  const previewMutate = previewMutation.mutate
  const previewPending = previewMutation.isPending
  const previewError = previewMutation.isError
  useEffect(() => {
    if (open && !preview && !previewPending && !previewError) {
      previewMutate(undefined, {
        onSuccess: (data) => {
          setPreview(data)
          if (data.movies.length > 0) {
            setActiveTab('movies')
          } else if (data.tvShows.length > 0) {
            setActiveTab('tv')
          }
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to generate preview')
        },
      })
    }
  }, [open, preview, previewPending, previewError, previewMutate])

  useEffect(() => {
    if (!open) {
      setPreview(null)
      setIsDebugData(false)
      setIsLoadingDebugData(false)
      setFilter('all')
      setSelectedFileIds(new Set())
      setManualEdits(new Map())
      setAssignModalOpen(false)
      previewMutation.reset()
    }
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleLoadDebugData = async () => {
    setIsLoadingDebugData(true)
    try {
      const debugPreview = await generateDebugPreview()
      setPreview(debugPreview)
      setIsDebugData(true)
      setActiveTab('movies')
      setFilter('all')
      setSelectedFileIds(new Set())
      setManualEdits(new Map())
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to generate debug data')
    } finally {
      setIsLoadingDebugData(false)
    }
  }

  // Apply manual edits to a file
  const applyEditToFile = useCallback(
    (file: FileMigrationPreview): FileMigrationPreview => {
      const edit = manualEdits.get(file.fileId)
      if (!edit) {
        return file
      }

      switch (edit.type) {
        case 'ignore': {
          return {
            ...file,
            proposedSlotId: null,
            proposedSlotName: undefined,
            needsReview: false,
            conflict: undefined,
            matchScore: 0,
          }
        }
        case 'assign': {
          return {
            ...file,
            proposedSlotId: edit.slotId,
            proposedSlotName: edit.slotName,
            needsReview: false,
            conflict: undefined,
            matchScore: 100,
          }
        }
        case 'unassign': {
          return {
            ...file,
            proposedSlotId: null,
            proposedSlotName: undefined,
            needsReview: true,
            conflict: undefined,
            matchScore: 0,
          }
        }
        default: {
          return file
        }
      }
    },
    [manualEdits],
  )

  // Edited preview with manual edits applied
  const editedPreview = useMemo((): MigrationPreview | null => {
    if (!preview) {
      return null
    }
    if (manualEdits.size === 0) {
      return preview
    }

    const editedMovies = preview.movies.map((movie) => ({
      ...movie,
      files: movie.files.map(applyEditToFile),
      hasConflict: movie.files.some((f) => {
        const edited = applyEditToFile(f)
        return !!edited.conflict || edited.needsReview
      }),
    }))

    const editedTvShows = preview.tvShows.map((show) => ({
      ...show,
      seasons: show.seasons.map((season) => ({
        ...season,
        episodes: season.episodes.map((episode) => ({
          ...episode,
          files: episode.files.map(applyEditToFile),
          hasConflict: episode.files.some((f) => {
            const edited = applyEditToFile(f)
            return !!edited.conflict || edited.needsReview
          }),
        })),
        hasConflict: season.episodes.some((e) =>
          e.files.some((f) => {
            const edited = applyEditToFile(f)
            return !!edited.conflict || edited.needsReview
          }),
        ),
      })),
      hasConflict: show.seasons.some((s) =>
        s.episodes.some((e) =>
          e.files.some((f) => {
            const edited = applyEditToFile(f)
            return !!edited.conflict || edited.needsReview
          }),
        ),
      ),
    }))

    // Recalculate summary
    const allMovieFiles = editedMovies.flatMap((m) => m.files)
    const allTvFiles = editedTvShows.flatMap((s) =>
      s.seasons.flatMap((se) => se.episodes.flatMap((e) => e.files)),
    )
    const allFiles = [...allMovieFiles, ...allTvFiles]

    const summary = {
      totalMovies: editedMovies.length,
      totalTvShows: editedTvShows.length,
      totalFiles: allFiles.length,
      filesWithSlots: allFiles.filter(
        (f) => f.proposedSlotId !== null && !f.needsReview && !f.conflict,
      ).length,
      filesNeedingReview: allFiles.filter((f) => f.needsReview && !f.conflict).length,
      conflicts: allFiles.filter((f) => !!f.conflict).length,
    }

    return { movies: editedMovies, tvShows: editedTvShows, summary }
  }, [preview, manualEdits, applyEditToFile])

  // Get all visible file IDs based on current tab and filter (used for Select All)
  const visibleFileIds = useMemo((): number[] => {
    if (!editedPreview) {
      return []
    }

    if (activeTab === 'movies') {
      let movies = editedPreview.movies
      switch (filter) {
        case 'assigned': {
          movies = movies.filter((movie) =>
            movie.files.every(
              (file) => file.proposedSlotId !== null && !file.needsReview && !file.conflict,
            ),
          )

          break
        }
        case 'conflicts': {
          movies = movies.filter((movie) => movie.files.some((file) => !!file.conflict))

          break
        }
        case 'nomatch': {
          return movies.flatMap((movie) =>
            movie.files.filter((f) => f.needsReview && !f.conflict).map((f) => f.fileId),
          )
        }
        // No default
      }
      return movies.flatMap((m) => m.files.map((f) => f.fileId))
    }
    const shows = editedPreview.tvShows
    if (filter === 'assigned') {
      return shows.flatMap((show) =>
        show.seasons.flatMap((season) =>
          season.episodes
            .filter((episode) =>
              episode.files.every(
                (file) => file.proposedSlotId !== null && !file.needsReview && !file.conflict,
              ),
            )
            .flatMap((episode) => episode.files.map((f) => f.fileId)),
        ),
      )
    }
    if (filter === 'conflicts') {
      return shows.flatMap((show) =>
        show.seasons.flatMap((season) =>
          season.episodes
            .filter((episode) => episode.files.some((file) => !!file.conflict))
            .flatMap((episode) => episode.files.map((f) => f.fileId)),
        ),
      )
    }
    if (filter === 'nomatch') {
      return shows.flatMap((show) =>
        show.seasons.flatMap((season) =>
          season.episodes.flatMap((episode) =>
            episode.files.filter((f) => f.needsReview && !f.conflict).map((f) => f.fileId),
          ),
        ),
      )
    }
    return shows.flatMap((s) =>
      s.seasons.flatMap((se) => se.episodes.flatMap((e) => e.files.map((f) => f.fileId))),
    )
  }, [editedPreview, activeTab, filter])

  // Compute ignored file IDs from manual edits
  const ignoredFileIds = useMemo((): Set<number> => {
    const ignored = new Set<number>()
    manualEdits.forEach((edit, fileId) => {
      if (edit.type === 'ignore') {
        ignored.add(fileId)
      }
    })
    return ignored
  }, [manualEdits])

  // Action handlers
  const handleToggleSelectAll = useCallback(() => {
    const allSelected = visibleFileIds.every((id) => selectedFileIds.has(id))
    if (allSelected) {
      setSelectedFileIds((prev) => {
        const next = new Set(prev)
        visibleFileIds.forEach((id) => next.delete(id))
        return next
      })
    } else {
      setSelectedFileIds((prev) => {
        const next = new Set(prev)
        visibleFileIds.forEach((id) => next.add(id))
        return next
      })
    }
  }, [visibleFileIds, selectedFileIds])

  const handleIgnore = useCallback(() => {
    setManualEdits((prev) => {
      const next = new Map(prev)
      selectedFileIds.forEach((fileId) => next.set(fileId, { type: 'ignore' }))
      return next
    })
    setSelectedFileIds(new Set())
  }, [selectedFileIds])

  const handleUnassign = useCallback(() => {
    setManualEdits((prev) => {
      const next = new Map(prev)
      selectedFileIds.forEach((fileId) => next.set(fileId, { type: 'unassign' }))
      return next
    })
    setSelectedFileIds(new Set())
  }, [selectedFileIds])

  const handleAssign = useCallback(
    (slotId: number, slotName: string) => {
      setManualEdits((prev) => {
        const next = new Map(prev)
        selectedFileIds.forEach((fileId) => next.set(fileId, { type: 'assign', slotId, slotName }))
        return next
      })
      setSelectedFileIds(new Set())
      setAssignModalOpen(false)
    },
    [selectedFileIds],
  )

  const handleReset = useCallback(() => {
    setManualEdits(new Map())
    setSelectedFileIds(new Set())
  }, [])

  const handleToggleFileSelection = useCallback((fileId: number) => {
    setSelectedFileIds((prev) => {
      const next = new Set(prev)
      if (next.has(fileId)) {
        next.delete(fileId)
      } else {
        next.add(fileId)
      }
      return next
    })
  }, [])

  const enabledSlots = useMemo(() => {
    return slots.filter((s) => s.enabled)
  }, [slots])

  // Filter movies based on selected filter (using edited preview)
  const filteredMovies = useMemo(() => {
    if (!editedPreview) {
      return []
    }
    if (filter === 'all') {
      return editedPreview.movies
    }

    // For "assigned" filter, only show movies where ALL files are cleanly assigned
    if (filter === 'assigned') {
      return editedPreview.movies.filter((movie) =>
        movie.files.every(
          (file) => file.proposedSlotId !== null && !file.needsReview && !file.conflict,
        ),
      )
    }

    // For "conflicts" filter, show movies that have ANY file with a conflict - but show ALL files for that movie
    if (filter === 'conflicts') {
      return editedPreview.movies.filter((movie) => movie.files.some((file) => !!file.conflict))
    }

    // For "nomatch" filter, show movies with files that have no match (needsReview but NOT a conflict)
    // filter === 'nomatch'
    return editedPreview.movies
      .map((movie) => {
        const filteredFiles = movie.files.filter((file) => file.needsReview && !file.conflict)
        if (filteredFiles.length === 0) {
          return null
        }
        return { ...movie, files: filteredFiles }
      })
      .filter((m): m is MovieMigrationPreview => m !== null)
  }, [editedPreview, filter])

  // Filter TV shows based on selected filter
  const filteredTvShows = useMemo(() => {
    if (!editedPreview) {
      return []
    }
    if (filter === 'all') {
      return editedPreview.tvShows
    }

    // For "assigned" filter, only show episodes where ALL files are cleanly assigned
    if (filter === 'assigned') {
      return editedPreview.tvShows
        .map((show) => {
          const filteredSeasons = show.seasons
            .map((season) => {
              const filteredEpisodes = season.episodes.filter((episode) =>
                episode.files.every(
                  (file) => file.proposedSlotId !== null && !file.needsReview && !file.conflict,
                ),
              )
              if (filteredEpisodes.length === 0) {
                return null
              }
              return {
                ...season,
                episodes: filteredEpisodes,
                totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
              }
            })
            .filter((s): s is NonNullable<typeof s> => s !== null)
          if (filteredSeasons.length === 0) {
            return null
          }
          return {
            ...show,
            seasons: filteredSeasons,
            totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
          }
        })
        .filter((s): s is TVShowMigrationPreview => s !== null)
    }

    // For "conflicts" filter, show episodes that have ANY file with a conflict - but show ALL files for that episode
    if (filter === 'conflicts') {
      return editedPreview.tvShows
        .map((show) => {
          const filteredSeasons = show.seasons
            .map((season) => {
              const filteredEpisodes = season.episodes.filter((episode) =>
                episode.files.some((file) => !!file.conflict),
              )
              if (filteredEpisodes.length === 0) {
                return null
              }
              return {
                ...season,
                episodes: filteredEpisodes,
                totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
              }
            })
            .filter((s): s is NonNullable<typeof s> => s !== null)
          if (filteredSeasons.length === 0) {
            return null
          }
          return {
            ...show,
            seasons: filteredSeasons,
            totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
          }
        })
        .filter((s): s is TVShowMigrationPreview => s !== null)
    }

    // For "nomatch" filter, show episodes with files that have no match (needsReview but NOT a conflict)
    // filter === 'nomatch'
    return editedPreview.tvShows
        .map((show) => {
          const filteredSeasons = show.seasons
            .map((season) => {
              const filteredEpisodes = season.episodes
                .map((episode) => {
                  const filteredFiles = episode.files.filter(
                    (file) => file.needsReview && !file.conflict,
                  )
                  if (filteredFiles.length === 0) {
                    return null
                  }
                  return { ...episode, files: filteredFiles }
                })
                .filter((e): e is NonNullable<typeof e> => e !== null)
              if (filteredEpisodes.length === 0) {
                return null
              }
              return {
                ...season,
                episodes: filteredEpisodes,
                totalFiles: filteredEpisodes.reduce((sum, e) => sum + e.files.length, 0),
              }
            })
            .filter((s): s is NonNullable<typeof s> => s !== null)
          if (filteredSeasons.length === 0) {
            return null
          }
          return {
            ...show,
            seasons: filteredSeasons,
            totalFiles: filteredSeasons.reduce((sum, s) => sum + s.totalFiles, 0),
          }
        })
        .filter((s): s is TVShowMigrationPreview => s !== null)
  }, [editedPreview, filter])

  const handleExecute = () => {
    // Convert manual edits to FileOverride array for the backend
    const overrides = [...manualEdits.entries()].map(([fileId, edit]) => ({
      fileId,
      type: edit.type,
      slotId: edit.type === 'assign' ? edit.slotId : undefined,
    }))

    executeMutation.mutate(overrides.length > 0 ? { overrides } : undefined, {
      onSuccess: (result) => {
        if (result.success) {
          toast.success(`Migration complete: ${result.filesAssigned} files assigned`)
          onOpenChange(false)
          onMigrationComplete()
        } else {
          toast.error('Migration completed with errors')
        }
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Migration failed')
      },
    })
  }

  const isLoading = previewMutation.isPending || isLoadingDebugData
  const isExecuting = executeMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Migration Dry Run Preview</DialogTitle>
          <DialogDescription>
            Review how your existing files will be assigned to version slots
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="text-muted-foreground size-8 animate-spin" />
            <span className="text-muted-foreground ml-3">
              {isLoadingDebugData
                ? 'Generating debug data with real slot matching...'
                : 'Analyzing library files...'}
            </span>
          </div>
        ) : editedPreview ? (
          <>
            {/* Summary - stays fixed, clickable to filter */}
            <div className="mb-2 grid shrink-0 grid-cols-4 gap-3">
              <SummaryCard
                label="All"
                value={editedPreview.summary.totalFiles}
                icon={FileVideo}
                active={filter === 'all'}
                onClick={() => setFilter('all')}
              />
              <SummaryCard
                label="Will Be Assigned"
                value={editedPreview.summary.filesWithSlots}
                icon={Check}
                variant="success"
                active={filter === 'assigned'}
                onClick={() => setFilter('assigned')}
              />
              <SummaryCard
                label="Conflicts"
                value={editedPreview.summary.conflicts}
                icon={AlertTriangle}
                variant={editedPreview.summary.conflicts > 0 ? 'warning' : 'default'}
                active={filter === 'conflicts'}
                onClick={() => setFilter('conflicts')}
              />
              <SummaryCard
                label="No Match"
                value={editedPreview.summary.filesNeedingReview}
                icon={HelpCircle}
                variant={editedPreview.summary.filesNeedingReview > 0 ? 'error' : 'default'}
                active={filter === 'nomatch'}
                onClick={() => setFilter('nomatch')}
              />
            </div>

            {/* Tab selector and action buttons */}
            <div className="mb-2 flex shrink-0 items-center justify-between border-b pb-2">
              <div className="flex gap-2">
                <Button
                  variant={activeTab === 'movies' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setActiveTab('movies')}
                  disabled={editedPreview.movies.length === 0}
                >
                  <Film className="mr-2 size-4" />
                  Movies ({editedPreview.movies.length})
                </Button>
                <Button
                  variant={activeTab === 'tv' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setActiveTab('tv')}
                  disabled={editedPreview.tvShows.length === 0}
                >
                  <Tv className="mr-2 size-4" />
                  Series ({editedPreview.tvShows.length})
                </Button>
              </div>

              {/* Action buttons */}
              <div className="flex gap-1.5">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleToggleSelectAll}
                  disabled={visibleFileIds.length === 0}
                >
                  {visibleFileIds.length > 0 &&
                  visibleFileIds.every((id) => selectedFileIds.has(id)) ? (
                    <CheckSquare className="mr-1.5 size-4" />
                  ) : (
                    <Square className="mr-1.5 size-4" />
                  )}
                  {visibleFileIds.length > 0 &&
                  visibleFileIds.every((id) => selectedFileIds.has(id))
                    ? 'Deselect All'
                    : 'Select All'}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleIgnore}
                  disabled={selectedFileIds.size === 0}
                >
                  <Ban className="mr-1.5 size-4" />
                  Ignore{selectedFileIds.size > 0 && ` (${selectedFileIds.size})`}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setAssignModalOpen(true)}
                  disabled={selectedFileIds.size === 0}
                >
                  <Layers className="mr-1.5 size-4" />
                  Assign...
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleUnassign}
                  disabled={selectedFileIds.size === 0}
                >
                  <XCircle className="mr-1.5 size-4" />
                  Unassign
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleReset}
                  disabled={manualEdits.size === 0}
                >
                  <RotateCcw className="mr-1.5 size-4" />
                  Reset
                </Button>
              </div>
            </div>

            {/* Scrollable content area */}
            <ScrollArea className="h-[50vh]">
              {activeTab === 'movies' ? (
                <MoviesList
                  movies={filteredMovies}
                  selectedFileIds={selectedFileIds}
                  ignoredFileIds={ignoredFileIds}
                  onToggleFileSelection={handleToggleFileSelection}
                />
              ) : (
                <TVShowsList
                  shows={filteredTvShows}
                  selectedFileIds={selectedFileIds}
                  ignoredFileIds={ignoredFileIds}
                  onToggleFileSelection={handleToggleFileSelection}
                />
              )}
            </ScrollArea>

            {/* Assign Modal */}
            <AssignModal
              open={assignModalOpen}
              onOpenChange={setAssignModalOpen}
              slots={enabledSlots}
              selectedCount={selectedFileIds.size}
              onAssign={handleAssign}
            />
          </>
        ) : (
          <div className="text-muted-foreground flex items-center justify-center py-16">
            Failed to load preview
          </div>
        )}

        <DialogFooter className="mt-2 shrink-0">
          <div className="flex flex-1 items-center gap-2">
            {developerMode ? (
              <Button
                variant="outline"
                size="sm"
                onClick={handleLoadDebugData}
                disabled={isExecuting || isLoadingDebugData}
                className="border-orange-300 text-orange-600 hover:bg-orange-50 dark:border-orange-700 dark:text-orange-400 dark:hover:bg-orange-950/50"
              >
                {isLoadingDebugData ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <Bug className="mr-2 size-4" />
                )}
                {isLoadingDebugData ? 'Loading...' : 'Load Debug Data'}
              </Button>
            ) : null}
            {isDebugData ? (
              <Badge
                variant="outline"
                className="border-orange-300 text-orange-600 dark:text-orange-400"
              >
                Debug Mode
              </Badge>
            ) : null}
          </div>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isExecuting}>
            Cancel
          </Button>
          <Button
            onClick={handleExecute}
            disabled={isLoading || isExecuting || !editedPreview || isDebugData}
          >
            {isExecuting ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            {isExecuting ? 'Executing...' : 'Execute Migration'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
