import { useCallback, useMemo, useState } from 'react'

import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'

import {
  useAssignMovieFile,
  useDeleteMovie,
  useExtendedMovieMetadata,
  useMovie,
  useMovieSlotStatus,
  useMultiVersionSettings,
  useQualityProfiles,
  useRefreshMovie,
  useSetMovieSlotMonitored,
  useSlots,
  useUpdateMovie,
} from '@/hooks'
import { movieKeys } from '@/hooks/use-movies'
import { useUIStore } from '@/stores'

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

  const getSlotName = useCallback((slotId: number | undefined) => {
    if (!slotId) {
      return null
    }
    return slots?.find((s) => s.id === slotId)?.name ?? null
  }, [slots])

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

  const globalLoading = useUIStore((s) => s.globalLoading)
  const queryClient = useQueryClient()
  const { data: movie, isLoading: queryLoading, isError, refetch } = useMovie(movieId)
  const isLoading = queryLoading || globalLoading
  const cachedMovie = queryClient.getQueryData<{ tmdbId?: number }>(movieKeys.detail(movieId))
  const tmdbId = movie?.tmdbId ?? cachedMovie?.tmdbId ?? 0
  const { data: extendedData, isLoading: isExtendedDataLoading } = useExtendedMovieMetadata(tmdbId)
  const { data: qualityProfiles } = useQualityProfiles('movie')
  const updateMutation = useUpdateMovie()
  const deleteMutation = useDeleteMovie()
  const refreshMutation = useRefreshMovie()
  const slotData = useSlotData(movieId)
  const qualityProfileName = useMemo(
    () => qualityProfiles?.find((p) => p.id === movie?.qualityProfileId)?.name,
    [qualityProfiles, movie?.qualityProfileId],
  )

  const handlers = createHandlers({
    movie, movieId, navigate,
    updateMutation, deleteMutation, refreshMutation,
    setSlotMonitoredMutation: slotData.setSlotMonitoredMutation,
    assignFileMutation: slotData.assignFileMutation,
    refetch: () => void refetch(),
  })

  return {
    movie, movieId, isLoading, isError, refetch,
    extendedData, isExtendedDataLoading, qualityProfileName,
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
