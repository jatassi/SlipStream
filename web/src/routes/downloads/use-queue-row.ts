import { toast } from 'sonner'

import type { ActionItem } from '@/components/presenter'
import {
  useFastForwardQueueItem,
  useMovie,
  usePauseQueueItem,
  useRemoveFromQueue,
  useResumeQueueItem,
  useSeriesDetail,
} from '@/hooks'
import { formatBytes, formatEta, formatSpeed } from '@/lib/formatters'
import type { QueueItem } from '@/types'

const STATE_LABEL: Partial<Record<string, string>> = {
  queued: 'Queued',
  completed: 'Importing',
  failed: 'Failed',
  warning: 'Warning',
}

function episodeLabelFor(item: QueueItem, movieYear?: number): string {
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

function stateLabelFor(item: QueueItem): string {
  return STATE_LABEL[item.status] ?? item.status
}

function statsLineFor(item: QueueItem): string {
  if (item.status === 'downloading') {
    return `${item.progress.toFixed(1)}% · ${formatSpeed(item.downloadSpeed)} · ${formatEta(item.eta)} left`
  }
  if (item.status === 'paused') {
    return `${item.progress.toFixed(1)}% · Paused · ${formatBytes(item.size)}`
  }
  return `${stateLabelFor(item)} · ${formatBytes(item.size)}`
}

async function runMutation(mutationFn: () => Promise<unknown>, successMsg: string) {
  await mutationFn()
  toast.success(successMsg)
}

type QueueOps = {
  isPaused: boolean
  canToggle: boolean
  togglePending: boolean
  toggle: () => void
  fastForward: () => void
  remove: (deleteFiles: boolean) => void
}

function useQueueOps(item: QueueItem): QueueOps {
  const removeMutation = useRemoveFromQueue()
  const pauseMutation = usePauseQueueItem()
  const resumeMutation = useResumeQueueItem()
  const fastForwardMutation = useFastForwardQueueItem()

  const clientItem = { clientId: item.clientId, id: item.id }
  const isPaused = item.status === 'paused'

  return {
    isPaused,
    canToggle: item.status === 'downloading' || isPaused,
    togglePending: pauseMutation.isPending || resumeMutation.isPending,
    toggle: () => {
      if (isPaused) {
        void runMutation(() => resumeMutation.mutateAsync(clientItem), 'Download resumed')
        return
      }
      void runMutation(() => pauseMutation.mutateAsync(clientItem), 'Download paused')
    },
    fastForward: () =>
      void runMutation(() => fastForwardMutation.mutateAsync(clientItem), 'Download completed'),
    remove: (deleteFiles: boolean) =>
      void runMutation(
        () => removeMutation.mutateAsync({ ...clientItem, deleteFiles }),
        deleteFiles ? 'Download removed with files' : 'Download removed',
      ),
  }
}

function buildActions(item: QueueItem, ops: QueueOps): ActionItem[] {
  const actions: ActionItem[] = []
  if (ops.canToggle) {
    actions.push({ label: ops.isPaused ? 'Resume' : 'Pause', onClick: ops.toggle })
  }
  if (item.clientType === 'mock') {
    actions.push({ label: 'Fast forward', onClick: ops.fastForward })
  }
  actions.push({
    label: 'Remove from queue',
    destructive: true,
    confirm: {
      title: 'Remove from queue',
      description: `Remove "${item.title}" from the queue?`,
      actions: [
        { label: 'Remove', destructive: true, onClick: () => ops.remove(false) },
        { label: 'Remove and delete files', destructive: true, onClick: () => ops.remove(true) },
      ],
    },
  })
  return actions
}

function movieIdOf(item: QueueItem): number {
  return item.mediaType === 'movie' && item.movieId ? item.movieId : 0
}

function seriesIdOf(item: QueueItem): number {
  return item.mediaType === 'series' && item.seriesId ? item.seriesId : 0
}

export function useQueueRow(item: QueueItem) {
  const ops = useQueueOps(item)
  const { data: movie } = useMovie(movieIdOf(item))
  const { data: series } = useSeriesDetail(seriesIdOf(item))

  return {
    tmdbId: item.mediaType === 'movie' ? movie?.tmdbId : series?.tmdbId,
    tvdbId: item.mediaType === 'series' ? series?.tvdbId : undefined,
    episodeLabel: episodeLabelFor(item, movie?.year),
    statsLine: statsLineFor(item),
    stateLabel: stateLabelFor(item),
    isPaused: ops.isPaused,
    canToggle: ops.canToggle,
    togglePending: ops.togglePending,
    toggle: ops.toggle,
    actions: buildActions(item, ops),
  }
}
