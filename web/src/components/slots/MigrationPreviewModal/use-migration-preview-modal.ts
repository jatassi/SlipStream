import { useCallback, useMemo } from 'react'

import { useDeveloperMode, useSlots } from '@/hooks'

import { computeEditedPreview } from '../shared/edit-utils'
import { filterMovies, filterTvShows, getVisibleMovieFileIds, getVisibleTvFileIds } from '../shared/filter-utils'
import type { DryRunModalProps } from '../shared/types'
import { useExecuteHandler } from './use-execute-migration'
import { useFileActions } from './use-file-actions'
import { usePreviewData } from './use-preview-data'

function useVisibleFileIds(
  editedPreview: ReturnType<typeof computeEditedPreview>,
  activeTab: 'movies' | 'tv',
  filter: 'all' | 'assigned' | 'conflicts' | 'nomatch',
) {
  return useMemo((): number[] => {
    if (!editedPreview) {
      return []
    }
    if (activeTab === 'movies') {
      return getVisibleMovieFileIds(editedPreview, filter)
    }
    return getVisibleTvFileIds(editedPreview, filter)
  }, [editedPreview, activeTab, filter])
}

export function useMigrationPreviewModal({ open, onOpenChange, onMigrationComplete }: DryRunModalProps) {
  const developerMode = useDeveloperMode()
  const { data: slots = [] } = useSlots()
  const pd = usePreviewData(open)
  const fa = useFileActions(open)
  const exec = useExecuteHandler(fa.manualEdits, onOpenChange, onMigrationComplete)
  const editedPreview = useMemo(
    () => computeEditedPreview(pd.preview, fa.manualEdits),
    [pd.preview, fa.manualEdits],
  )
  const visibleFileIds = useVisibleFileIds(editedPreview, pd.activeTab, pd.filter)
  const allSelected = visibleFileIds.length > 0 && visibleFileIds.every((id) => fa.selectedFileIds.has(id))
  const handleToggleSelectAll = useCallback(() => {
    fa.setSelectedFileIds((prev) => {
      const next = new Set(prev)
      for (const id of visibleFileIds) {
        if (allSelected) { next.delete(id) } else { next.add(id) }
      }
      return next
    })
  }, [visibleFileIds, allSelected, fa])

  return {
    editedPreview, visibleFileIds, allSelected, developerMode,
    filteredMovies: useMemo(() => filterMovies(editedPreview, pd.filter), [editedPreview, pd.filter]),
    filteredTvShows: useMemo(() => filterTvShows(editedPreview, pd.filter), [editedPreview, pd.filter]),
    enabledSlots: useMemo(() => slots.filter((s) => s.enabled), [slots]),
    activeTab: pd.activeTab, setActiveTab: pd.setActiveTab,
    filter: pd.filter, setFilter: pd.setFilter,
    isDebugData: pd.isDebugData, isLoadingDebugData: pd.isLoadingDebugData,
    isLoading: pd.isLoading, isExecuting: exec.isExecuting,
    selectedFileIds: fa.selectedFileIds, ignoredFileIds: fa.ignoredFileIds,
    assignModalOpen: fa.assignModalOpen, setAssignModalOpen: fa.setAssignModalOpen,
    manualEditsCount: fa.manualEdits.size, selectedCount: fa.selectedFileIds.size,
    handleToggleSelectAll, handleExecute: exec.handleExecute,
    handleIgnore: fa.handleIgnore, handleUnassign: fa.handleUnassign,
    handleAssign: fa.handleAssign, handleReset: fa.handleReset,
    handleToggleFileSelection: fa.handleToggleFileSelection,
    handleLoadDebugData: pd.handleLoadDebugData,
  }
}
