import type { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'

import type {
  useAssignMovieFile,
  useDeleteMovie,
  useRefreshMovie,
  useSetMovieSlotMonitored,
  useUpdateMovie,
} from '@/hooks'
import type { Movie } from '@/types'

type HandlerDeps = {
  movie: Movie | undefined
  movieId: number
  navigate: ReturnType<typeof useNavigate>
  updateMutation: ReturnType<typeof useUpdateMovie>
  deleteMutation: ReturnType<typeof useDeleteMovie>
  refreshMutation: ReturnType<typeof useRefreshMovie>
  setSlotMonitoredMutation: ReturnType<typeof useSetMovieSlotMonitored>
  assignFileMutation: ReturnType<typeof useAssignMovieFile>
  refetch: () => void
}

function createMovieHandlers(deps: HandlerDeps) {
  const handleToggleMonitored = async (newMonitored?: boolean) => {
    if (!deps.movie) {
      return
    }
    const target = newMonitored ?? !deps.movie.monitored
    try {
      await deps.updateMutation.mutateAsync({
        id: deps.movie.id,
        data: { monitored: target },
      })
      toast.success(target ? 'Movie monitored' : 'Movie unmonitored')
    } catch {
      toast.error('Failed to update movie')
    }
  }

  const handleRefresh = async () => {
    try {
      await deps.refreshMutation.mutateAsync(deps.movieId)
      toast.success('Metadata refreshed')
    } catch {
      toast.error('Failed to refresh metadata')
    }
  }

  const handleDelete = async () => {
    try {
      await deps.deleteMutation.mutateAsync({ id: deps.movieId })
      toast.success('Movie deleted')
      void deps.navigate({ to: '/movies' })
    } catch {
      toast.error('Failed to delete movie')
    }
  }

  return { handleToggleMonitored, handleRefresh, handleDelete }
}

function createSlotHandlers(deps: HandlerDeps) {
  const handleToggleSlotMonitored = async (slotId: number, monitored: boolean) => {
    try {
      await deps.setSlotMonitoredMutation.mutateAsync({
        movieId: deps.movieId,
        slotId,
        data: { monitored },
      })
      toast.success(monitored ? 'Slot monitored' : 'Slot unmonitored')
    } catch {
      toast.error('Failed to update slot monitoring')
    }
  }

  const handleAssignFileToSlot = async (fileId: number, slotId: number) => {
    try {
      await deps.assignFileMutation.mutateAsync({
        movieId: deps.movieId,
        slotId,
        data: { fileId },
      })
      deps.refetch()
      toast.success('File assigned to slot')
    } catch {
      toast.error('Failed to assign file to slot')
    }
  }

  return { handleToggleSlotMonitored, handleAssignFileToSlot }
}

export function createHandlers(deps: HandlerDeps) {
  return {
    ...createMovieHandlers(deps),
    ...createSlotHandlers(deps),
  }
}
