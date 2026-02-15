import { useMemo } from 'react'

import { computeEditedPreview } from '../shared/edit-utils'
import type { FilterType } from '../shared/filter-utils'
import { filterMovies, filterTvShows, getVisibleMovieFileIds, getVisibleTvFileIds } from '../shared/filter-utils'
import type { DryRunModalProps, MigrationPreview } from '../shared/types'
import { useModalActions } from './use-modal-actions'
import { useModalState } from './use-modal-state'

function useVisibleFileIds(editedPreview: MigrationPreview | null, activeTab: string, filter: FilterType) {
  return useMemo((): number[] => {
    if (!editedPreview) { return [] }
    if (activeTab === 'movies') { return getVisibleMovieFileIds(editedPreview, filter) }
    return getVisibleTvFileIds(editedPreview, filter)
  }, [editedPreview, activeTab, filter])
}

function useFilteredMedia(editedPreview: MigrationPreview | null, filter: FilterType) {
  const movies = useMemo(
    () => (editedPreview ? filterMovies(editedPreview, filter) : []),
    [editedPreview, filter],
  )
  const tvShows = useMemo(
    () => (editedPreview ? filterTvShows(editedPreview, filter) : []),
    [editedPreview, filter],
  )
  return { filteredMovies: movies, filteredTvShows: tvShows }
}

export function useDryRunModal(props: DryRunModalProps) {
  const { s, setS, patch, ignoredFileIds, isLoading, developerMode } = useModalState(props.open)

  const editedPreview = useMemo(
    () => (s.preview ? computeEditedPreview(s.preview, s.manualEdits) : null),
    [s.preview, s.manualEdits],
  )

  const visibleFileIds = useVisibleFileIds(editedPreview, s.activeTab, s.filter)
  const actions = useModalActions({
    setS, patch, props, visibleFileIds, selectedFileIds: s.selectedFileIds, manualEdits: s.manualEdits,
  })
  const enabledSlots = useMemo(() => actions.slots.filter((sl) => sl.enabled), [actions.slots])
  const { filteredMovies, filteredTvShows } = useFilteredMedia(editedPreview, s.filter)

  const allFilesAccountedFor = editedPreview
    ? editedPreview.summary.filesWithSlots + ignoredFileIds.size === editedPreview.summary.totalFiles
    : false

  return {
    editedPreview,
    activeTab: s.activeTab,
    setActiveTab: (tab: 'movies' | 'tv') => { patch({ activeTab: tab }) },
    filter: s.filter,
    setFilter: (f: FilterType) => { patch({ filter: f }) },
    selectedFileIds: s.selectedFileIds,
    ignoredFileIds,
    visibleFileIds,
    filteredMovies,
    filteredTvShows,
    enabledSlots,
    assignModalOpen: s.assignModalOpen,
    setAssignModalOpen: (v: boolean) => { patch({ assignModalOpen: v }) },
    confirmModalOpen: s.confirmModalOpen,
    setConfirmModalOpen: (v: boolean) => { patch({ confirmModalOpen: v }) },
    isLoading,
    isLoadingDebugData: s.isLoadingDebugData,
    isDebugData: s.isDebugData,
    developerMode,
    allFilesAccountedFor,
    manualEditsCount: s.manualEdits.size,
    ...actions,
  }
}
