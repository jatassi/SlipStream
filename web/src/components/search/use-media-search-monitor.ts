import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from 'react'

import { toast } from 'sonner'

import {
  useAutoSearchEpisode,
  useAutoSearchEpisodeSlot,
  useAutoSearchMovie,
  useAutoSearchMovieSlot,
  useAutoSearchSeason,
  useAutoSearchSeries,
  useDeveloperMode,
  useMediaDownloadProgress,
} from '@/hooks'

import type {
  ControlState,
  MediaSearchMonitorControlsProps,
  ResolvedSize,
  SearchModalExternalProps,
} from './media-search-monitor-types'
import {
  buildDownloadTarget,
  buildSearchModalProps,
  formatBatchResult,
  formatSingleResult,
} from './media-search-monitor-utils'

const SM_BREAKPOINT = '(max-width: 819px)'
const smSubscribe = (cb: () => void) => {
  const mql = globalThis.matchMedia(SM_BREAKPOINT)
  mql.addEventListener('change', cb)
  return () => mql.removeEventListener('change', cb)
}
const smSnapshot = () => globalThis.matchMedia(SM_BREAKPOINT).matches
const smServer = () => false

function resolveSize(sizeProp: string, isSmall: boolean): ResolvedSize {
  if (sizeProp === 'responsive') {
    return isSmall ? 'sm' : 'lg'
  }
  return sizeProp as ResolvedSize
}

export function useMediaSearchMonitor(props: MediaSearchMonitorControlsProps) {
  const { title, size: sizeProp, qualityProfileId } = props

  const isSmall = useSyncExternalStore(smSubscribe, smSnapshot, smServer)
  const size = resolveSize(sizeProp, isSmall)

  const downloadTarget = buildDownloadTarget(props)
  const downloadProgress = useMediaDownloadProgress(downloadTarget)
  const { controlState, setControlState, completionTimerRef } = useDownloadStateTracking(downloadProgress.isDownloading)

  const developerMode = useDeveloperMode()
  const mutations = useAutoSearchMutations()

  const handlers = useControlHandlers({
    controlState,
    setControlState,
    completionTimerRef,
    downloadProgress,
    developerMode,
    mutations,
    props,
    title,
  })

  const searchModalProps: SearchModalExternalProps = buildSearchModalProps(props, qualityProfileId)
  const effectiveState: ControlState = downloadProgress.isDownloading
    ? { type: 'progress' }
    : controlState

  return {
    size,
    effectiveState,
    downloadProgress,
    searchModalProps,
    ...handlers,
  }
}

function useAutoSearchMutations() {
  return {
    movieMutation: useAutoSearchMovie(),
    episodeMutation: useAutoSearchEpisode(),
    seasonMutation: useAutoSearchSeason(),
    seriesMutation: useAutoSearchSeries(),
    movieSlotMutation: useAutoSearchMovieSlot(),
    episodeSlotMutation: useAutoSearchEpisodeSlot(),
  }
}

function useDownloadStateTracking(isDownloading: boolean) {
  const [controlState, setControlState] = useState<ControlState>({ type: 'default' })
  const completionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [prevDownloading, setPrevDownloading] = useState(isDownloading)

  if (isDownloading && controlState.type === 'default') {
    setControlState({ type: 'progress' })
  }
  if (isDownloading !== prevDownloading) {
    setPrevDownloading(isDownloading)
    if (isDownloading && controlState.type !== 'progress') {
      setControlState({ type: 'progress' })
    }
    if (!isDownloading && controlState.type === 'progress') {
      setControlState({ type: 'completed' })
    }
  }

  useEffect(() => {
    if (controlState.type !== 'completed') {
      return
    }
    const timer = setTimeout(() => {
      setControlState({ type: 'default' })
    }, 2500)
    completionTimerRef.current = timer
    return () => {
      clearTimeout(timer)
    }
  }, [controlState.type])

  return { controlState, setControlState, completionTimerRef }
}

type ControlHandlersDeps = {
  controlState: ControlState
  setControlState: (s: ControlState) => void
  completionTimerRef: React.RefObject<ReturnType<typeof setTimeout> | null>
  downloadProgress: ReturnType<typeof useMediaDownloadProgress>
  developerMode: boolean
  mutations: ReturnType<typeof useAutoSearchMutations>
  props: MediaSearchMonitorControlsProps
  title: string
}

function useControlHandlers(deps: ControlHandlersDeps) {
  const { controlState, setControlState, completionTimerRef, downloadProgress, developerMode, mutations, props, title } = deps
  const [searchModalOpen, setSearchModalOpen] = useState(false)

  const handleManualSearch = useCallback(() => {
    setSearchModalOpen(true)
    setControlState({ type: 'searching', mode: 'manual' })
  }, [setControlState])

  const handleModalClose = useCallback(
    (open: boolean) => {
      setSearchModalOpen(open)
      if (!open && controlState.type === 'searching' && controlState.mode === 'manual' && !downloadProgress.isDownloading) {
        setControlState({ type: 'default' })
      }
    },
    [controlState, downloadProgress.isDownloading, setControlState],
  )

  const handleGrabSuccess = useCallback(() => {
    setControlState({ type: 'progress' })
  }, [setControlState])

  const handleAutoSearch = useCallback(async () => {
    setControlState({ type: 'searching', mode: 'auto' })
    try {
      if (developerMode) {
        await new Promise((r) => setTimeout(r, 5000))
      }
      await dispatchAutoSearch(props, { ...mutations, title, setControlState })
    } catch (error) {
      handleAutoSearchError(error, title, setControlState)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props, title, developerMode, setControlState])

  const handleErrorDismiss = useCallback(() => {
    setControlState({ type: 'default' })
  }, [setControlState])

  const handleCompletionClick = useCallback(() => {
    if (completionTimerRef.current) {
      clearTimeout(completionTimerRef.current)
    }
    setControlState({ type: 'default' })
  }, [completionTimerRef, setControlState])

  return {
    searchModalOpen,
    handleManualSearch,
    handleModalClose,
    handleGrabSuccess,
    handleAutoSearch,
    handleErrorDismiss,
    handleCompletionClick,
  }
}

type AutoSearchDeps = {
  movieMutation: ReturnType<typeof useAutoSearchMovie>
  episodeMutation: ReturnType<typeof useAutoSearchEpisode>
  seasonMutation: ReturnType<typeof useAutoSearchSeason>
  seriesMutation: ReturnType<typeof useAutoSearchSeries>
  movieSlotMutation: ReturnType<typeof useAutoSearchMovieSlot>
  episodeSlotMutation: ReturnType<typeof useAutoSearchEpisodeSlot>
  title: string
  setControlState: (state: ControlState) => void
}

function applySearchResult(downloaded: boolean, found: boolean, set: (s: ControlState) => void) {
  if (downloaded) {
    set({ type: 'progress' })
  } else if (found) {
    set({ type: 'default' })
  } else {
    set({ type: 'error', message: 'Not Found' })
  }
}

function applyBatchResult(result: { downloaded: number; found: number; failed: number }, set: (s: ControlState) => void) {
  if (result.downloaded > 0) {
    set({ type: 'progress' })
  } else if (result.found === 0 && result.failed === 0) {
    set({ type: 'error', message: 'Not Found' })
  } else {
    set({ type: 'default' })
  }
}

async function dispatchAutoSearch(props: MediaSearchMonitorControlsProps, deps: AutoSearchDeps) {
  const { title, setControlState } = deps

  switch (props.mediaType) {
    case 'movie': {
      const result = await deps.movieMutation.mutateAsync(props.movieId)
      formatSingleResult(result, title)
      applySearchResult(result.downloaded, result.found, setControlState)
      break
    }
    case 'episode': {
      const result = await deps.episodeMutation.mutateAsync(props.episodeId)
      formatSingleResult(result, title)
      applySearchResult(result.downloaded, result.found, setControlState)
      break
    }
    case 'season': {
      const result = await deps.seasonMutation.mutateAsync({
        seriesId: props.seriesId,
        seasonNumber: props.seasonNumber,
      })
      formatBatchResult(result, `Season ${props.seasonNumber}`)
      applyBatchResult(result, setControlState)
      break
    }
    case 'series': {
      const result = await deps.seriesMutation.mutateAsync(props.seriesId)
      formatBatchResult(result, title)
      applyBatchResult(result, setControlState)
      break
    }
    case 'movie-slot': {
      const result = await deps.movieSlotMutation.mutateAsync({
        movieId: props.movieId,
        slotId: props.slotId,
      })
      applySlotResult(result, setControlState)
      break
    }
    case 'episode-slot': {
      const result = await deps.episodeSlotMutation.mutateAsync({
        episodeId: props.episodeId,
        slotId: props.slotId,
      })
      applySlotResult(result, setControlState)
      break
    }
  }
}

function applySlotResult(
  result: { downloaded: boolean; found: boolean },
  set: (s: ControlState) => void,
) {
  if (result.downloaded) {
    toast.success('Release grabbed for slot')
    set({ type: 'progress' })
  } else if (result.found) {
    toast.info('Release found but not grabbed')
    set({ type: 'default' })
  } else {
    toast.warning('No releases found')
    set({ type: 'error', message: 'Not Found' })
  }
}

function handleAutoSearchError(
  error: unknown,
  title: string,
  set: (s: ControlState) => void,
) {
  if (error instanceof Error && error.message.includes('409')) {
    toast.warning(`"${title}" is already in the download queue`)
    set({ type: 'progress' })
  } else {
    toast.error(`Search failed for "${title}"`)
    set({ type: 'error', message: 'Failed' })
  }
}
