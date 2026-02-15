import { useCallback, useEffect, useMemo, useState } from 'react'

import { toast } from 'sonner'

import { useDeveloperMode, useMigrationPreview } from '@/hooks'

import type { FilterType } from '../shared/filter-utils'
import type { ManualEdit, MigrationPreview } from '../shared/types'

export type ModalStateBundle = {
  preview: MigrationPreview | null
  activeTab: 'movies' | 'tv'
  isDebugData: boolean
  isLoadingDebugData: boolean
  filter: FilterType
  selectedFileIds: Set<number>
  manualEdits: Map<number, ManualEdit>
  assignModalOpen: boolean
  confirmModalOpen: boolean
}

const INITIAL_STATE: ModalStateBundle = {
  preview: null,
  activeTab: 'movies',
  isDebugData: false,
  isLoadingDebugData: false,
  filter: 'all',
  selectedFileIds: new Set(),
  manualEdits: new Map(),
  assignModalOpen: false,
  confirmModalOpen: false,
}

function resolveInitialTab(data: MigrationPreview): 'movies' | 'tv' {
  if (data.movies.length > 0) { return 'movies' }
  if (data.tvShows.length > 0) { return 'tv' }
  return 'movies'
}

export function useModalState(open: boolean) {
  const previewMutation = useMigrationPreview()
  const developerMode = useDeveloperMode()
  const [s, setS] = useState<ModalStateBundle>(INITIAL_STATE)

  const patch = useCallback(
    (u: Partial<ModalStateBundle>) => { setS((prev) => ({ ...prev, ...u })) },
    [],
  )

  const { mutate, isPending, isError } = previewMutation
  useEffect(() => {
    if (!open || s.preview || isPending || isError) { return }
    mutate(undefined, {
      onSuccess: (data) => { patch({ preview: data, activeTab: resolveInitialTab(data) }) },
      onError: (err) => { toast.error(err instanceof Error ? err.message : 'Failed to generate preview') },
    })
  }, [open, s.preview, isPending, isError, mutate, patch])

  useEffect(() => {
    if (!open) {
      setS({ ...INITIAL_STATE, selectedFileIds: new Set(), manualEdits: new Map() })
      previewMutation.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const ignoredFileIds = useMemo((): Set<number> => {
    const ignored = new Set<number>()
    s.manualEdits.forEach((edit, fileId) => {
      if (edit.type === 'ignore') { ignored.add(fileId) }
    })
    return ignored
  }, [s.manualEdits])

  return { s, setS, patch, ignoredFileIds, isLoading: isPending || s.isLoadingDebugData, developerMode }
}
