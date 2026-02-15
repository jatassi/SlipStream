import { useMemo, useState } from 'react'

import { useNavigate, useParams } from '@tanstack/react-router'

import {
  useAssignMovieFile,
  useDeleteMovie,
  useExtendedMovieMetadata,
  useGlobalLoading,
  useMovie,
  useMovieSlotStatus,
  useMultiVersionSettings,
  useQualityProfiles,
  useRefreshMovie,
  useSetMovieSlotMonitored,
  useSlots,
  useUpdateMovie,
} from '@/hooks'

import { createHandlers } from './movie-detail-handlers'

function useSlotData(movieId: number) {
  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slotStatus, isLoading: isLoadingSlotStatus } = useMovieSlotStatus(movieId)
  const { data: slots } = useSlots()
  const setSlotMonitoredMutation = useSetMovieSlotMonitored()
  const assignFileMutation = useAssignMovieFile()
  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = useMemo(() => slots?.filter((s) => s.enabled) ?? [], [slots])

  const slotQualityProfiles = useMemo(() => {
    const map: Record<number, number> = {}
    for (const slot of enabledSlots) {
      if (slot.qualityProfileId !== null) {
        map[slot.id] = slot.qualityProfileId
      }
    }
    return map
  }, [enabledSlots])

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) {
      return null
    }
    return slots?.find((s) => s.id === slotId)?.name ?? null
  }

  return {
    isMultiVersionEnabled, slotStatus, isLoadingSlotStatus,
    slotQualityProfiles, enabledSlots,
    setSlotMonitoredMutation, assignFileMutation, getSlotName,
  }
}

export function useMovieDetail() {
  const params: { id: string } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const movieId = Number.parseInt(params.id, 10)

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)
  const [expandedFileId, setExpandedFileId] = useState<number | null>(null)

  const globalLoading = useGlobalLoading()
  const { data: movie, isLoading: queryLoading, isError, refetch } = useMovie(movieId)
  const isLoading = queryLoading || globalLoading
  const { data: extendedData } = useExtendedMovieMetadata(movie?.tmdbId ?? 0)
  const { data: qualityProfiles } = useQualityProfiles()
  const updateMutation = useUpdateMovie()
  const deleteMutation = useDeleteMovie()
  const refreshMutation = useRefreshMovie()
  const slotData = useSlotData(movieId)
  const qualityProfileName = qualityProfiles?.find((p) => p.id === movie?.qualityProfileId)?.name

  const handlers = createHandlers({
    movie, movieId, navigate,
    updateMutation, deleteMutation, refreshMutation,
    setSlotMonitoredMutation: slotData.setSlotMonitoredMutation,
    assignFileMutation: slotData.assignFileMutation,
    refetch: () => void refetch(),
  })

  return {
    movie, movieId, isLoading, isError, refetch,
    extendedData, qualityProfileName,
    editDialogOpen, setEditDialogOpen,
    overviewExpanded,
    toggleOverviewExpanded: () => setOverviewExpanded((prev) => !prev),
    expandedFileId,
    toggleExpandedFile: (fileId: number) => setExpandedFileId((prev) => (prev === fileId ? null : fileId)),
    refreshMutation,
    ...slotData,
    ...handlers,
  }
}

export type MovieDetailState = ReturnType<typeof useMovieDetail>
