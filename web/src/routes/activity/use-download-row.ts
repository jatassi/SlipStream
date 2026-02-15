import { toast } from 'sonner'

import {
  useFastForwardQueueItem,
  useMovie,
  usePauseQueueItem,
  useRemoveFromQueue,
  useResumeQueueItem,
  useSeriesDetail,
} from '@/hooks'
import { formatBytes } from '@/lib/formatters'
import type { QueueItem } from '@/types'

function getTitleSuffix(item: QueueItem, movieYear?: number): string {
  if (item.mediaType === 'movie') {
    return movieYear ? `(${movieYear})` : ''
  }
  if (item.episode && item.season) {
    return `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
  }
  if (item.isSeasonPack && item.season) {
    return `S${String(item.season).padStart(2, '0')}`
  }
  if (item.isCompleteSeries) {
    return 'Complete Series'
  }
  return ''
}

function formatProgressText(downloadedSize: number, totalSize: number): string {
  const downloadedFormatted = formatBytes(downloadedSize)
  const totalFormatted = formatBytes(totalSize)
  const totalParts = /^([\d.]+)\s*(.+)$/.exec(totalFormatted)
  const downloadedParts = /^([\d.]+)\s*(.+)$/.exec(downloadedFormatted)

  if (totalParts && downloadedParts && totalParts[2] === downloadedParts[2]) {
    return `${downloadedParts[1]}/${totalParts[1]} ${totalParts[2]}`
  }
  return `${downloadedFormatted}/${totalFormatted}`
}

async function runMutation(
  mutationFn: () => Promise<unknown>,
  successMsg: string,
  errorMsg: string,
) {
  try {
    await mutationFn()
    toast.success(successMsg)
  } catch {
    toast.error(errorMsg)
  }
}

function getMovieId(item: QueueItem): number {
  if (item.mediaType === 'movie' && item.movieId) {
    return item.movieId
  }
  return 0
}

function getSeriesId(item: QueueItem): number {
  if (item.mediaType === 'series' && item.seriesId) {
    return item.seriesId
  }
  return 0
}

export function useDownloadRow(item: QueueItem) {
  const removeMutation = useRemoveFromQueue()
  const pauseMutation = usePauseQueueItem()
  const resumeMutation = useResumeQueueItem()
  const fastForwardMutation = useFastForwardQueueItem()

  const { data: movie } = useMovie(getMovieId(item))
  const { data: series } = useSeriesDetail(getSeriesId(item))

  const isMovie = item.mediaType === 'movie'
  const isSeries = item.mediaType === 'series'
  const clientItem = { clientId: item.clientId, id: item.id }
  const tmdbId = isMovie ? movie?.tmdbId : series?.tmdbId
  const tvdbId = isSeries ? series?.tvdbId : undefined

  return {
    isMovie,
    isSeries,
    tmdbId,
    tvdbId,
    titleSuffix: getTitleSuffix(item, movie?.year),
    progressText: formatProgressText(item.downloadedSize, item.size),
    handlePause: () => runMutation(() => pauseMutation.mutateAsync(clientItem), 'Download paused', 'Failed to pause download'),
    handleResume: () => runMutation(() => resumeMutation.mutateAsync(clientItem), 'Download resumed', 'Failed to resume download'),
    handleFastForward: () => runMutation(() => fastForwardMutation.mutateAsync(clientItem), 'Download completed', 'Failed to fast forward download'),
    handleRemove: (deleteFiles: boolean) =>
      runMutation(
        () => removeMutation.mutateAsync({ ...clientItem, deleteFiles }),
        deleteFiles ? 'Download removed with files' : 'Download removed',
        'Failed to remove download',
      ),
    pauseIsPending: pauseMutation.isPending,
    resumeIsPending: resumeMutation.isPending,
    fastForwardIsPending: fastForwardMutation.isPending,
  }
}
