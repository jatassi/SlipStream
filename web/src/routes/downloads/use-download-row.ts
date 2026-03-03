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

const RE_SIZE_PARTS = /^([\d.]+)\s*(.+)$/

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
  const totalParts = RE_SIZE_PARTS.exec(totalFormatted)
  const downloadedParts = RE_SIZE_PARTS.exec(downloadedFormatted)

  if (totalParts && totalParts[2] === downloadedParts?.[2]) {
    return `${downloadedParts[1]}/${totalParts[1]} ${totalParts[2]}`
  }
  return `${downloadedFormatted}/${totalFormatted}`
}

async function runMutation(
  mutationFn: () => Promise<unknown>,
  successMsg: string,
) {
  await mutationFn()
  toast.success(successMsg)
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
    handlePause: () => runMutation(() => pauseMutation.mutateAsync(clientItem), 'Download paused'),
    handleResume: () => runMutation(() => resumeMutation.mutateAsync(clientItem), 'Download resumed'),
    handleFastForward: () => runMutation(() => fastForwardMutation.mutateAsync(clientItem), 'Download completed'),
    handleRemove: (deleteFiles: boolean) =>
      runMutation(
        () => removeMutation.mutateAsync({ ...clientItem, deleteFiles }),
        deleteFiles ? 'Download removed with files' : 'Download removed',
      ),
    pauseIsPending: pauseMutation.isPending,
    resumeIsPending: resumeMutation.isPending,
    fastForwardIsPending: fastForwardMutation.isPending,
  }
}
