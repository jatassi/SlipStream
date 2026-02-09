import { useState, useRef, useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  UserSearch,
  Zap,
  Eye,
  EyeOff,
  AlertCircle,
  Check,
  Download,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { SearchModal } from './SearchModal'
import {
  useAutoSearchMovie,
  useAutoSearchEpisode,
  useAutoSearchSeason,
  useAutoSearchSeries,
  useAutoSearchMovieSlot,
  useAutoSearchEpisodeSlot,
  useMediaDownloadProgress,
  useDeveloperMode,
} from '@/hooks'
import type { MediaTarget } from '@/hooks/useMediaDownloadProgress'
import type { AutoSearchResult, BatchAutoSearchResult } from '@/types'
import { formatBytes, formatSpeed, formatEta } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

type MediaTheme = 'movie' | 'tv'
type ControlSize = 'lg' | 'sm' | 'xs' | 'responsive'

interface BaseProps {
  title: string
  theme: MediaTheme
  size: ControlSize
  monitored: boolean
  onMonitoredChange: (monitored: boolean) => void
  monitorDisabled?: boolean
  qualityProfileId: number
  className?: string
}

interface MovieProps extends BaseProps {
  mediaType: 'movie'
  movieId: number
  tmdbId?: number
  imdbId?: string
  year?: number
}

interface SeriesProps extends BaseProps {
  mediaType: 'series'
  seriesId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

interface SeasonProps extends BaseProps {
  mediaType: 'season'
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

interface EpisodeProps extends BaseProps {
  mediaType: 'episode'
  episodeId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

interface MovieSlotProps extends BaseProps {
  mediaType: 'movie-slot'
  movieId: number
  slotId: number
  tmdbId?: number
  imdbId?: string
  year?: number
}

interface EpisodeSlotProps extends BaseProps {
  mediaType: 'episode-slot'
  episodeId: number
  slotId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

export type MediaSearchMonitorControlsProps =
  | MovieProps
  | SeriesProps
  | SeasonProps
  | EpisodeProps
  | MovieSlotProps
  | EpisodeSlotProps

// ---------------------------------------------------------------------------
// State machine
// ---------------------------------------------------------------------------

type ControlState =
  | { type: 'default' }
  | { type: 'searching'; mode: 'manual' | 'auto' }
  | { type: 'progress' }
  | { type: 'completed' }
  | { type: 'error'; message: string }

// ---------------------------------------------------------------------------
// Auto-search result formatting
// ---------------------------------------------------------------------------

function formatSingleResult(result: AutoSearchResult, title: string): void {
  if (result.error) {
    toast.error(`Search failed for "${title}"`, { description: result.error })
    return
  }
  if (!result.found) {
    toast.warning(`No releases found for "${title}"`)
    return
  }
  if (result.downloaded) {
    const message = result.upgraded ? 'Quality upgrade found' : 'Found and downloading'
    toast.success(`${message}: ${result.release?.title || title}`, {
      description: result.clientName ? `Sent to ${result.clientName}` : undefined,
    })
  } else {
    toast.info(`Release found but not downloaded: ${result.release?.title || title}`)
  }
}

function formatBatchResult(result: BatchAutoSearchResult, title: string): void {
  if (result.downloaded > 0) {
    toast.success(`Found ${result.downloaded} releases for "${title}"`, {
      description: `Searched ${result.totalSearched} items`,
    })
  } else if (result.found > 0) {
    toast.info(`Found ${result.found} releases but none downloaded for "${title}"`)
  } else if (result.failed > 0) {
    toast.error(`Search failed for ${result.failed} items in "${title}"`)
  } else {
    toast.warning(`No releases found for "${title}"`)
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const SM_BREAKPOINT = '(max-width: 819px)'
const smSubscribe = (cb: () => void) => {
  const mql = window.matchMedia(SM_BREAKPOINT)
  mql.addEventListener('change', cb)
  return () => mql.removeEventListener('change', cb)
}
const smSnapshot = () => window.matchMedia(SM_BREAKPOINT).matches
const smServer = () => false

export function MediaSearchMonitorControls(props: MediaSearchMonitorControlsProps) {
  const {
    title,
    theme,
    size: sizeProp,
    monitored,
    onMonitoredChange,
    monitorDisabled,
    qualityProfileId,
    className,
  } = props

  const isSmall = useSyncExternalStore(smSubscribe, smSnapshot, smServer)
  const size: 'lg' | 'sm' | 'xs' = sizeProp === 'responsive' ? (isSmall ? 'sm' : 'lg') : sizeProp

  // ---- State ----
  const [controlState, setControlState] = useState<ControlState>({ type: 'default' })
  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const completionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // ---- Download progress ----
  const downloadTarget = buildDownloadTarget(props)
  const downloadProgress = useMediaDownloadProgress(downloadTarget)

  // Mount-time detection: if downloads already exist, go to progress immediately
  const prevDownloading = useRef(downloadProgress.isDownloading)
  if (downloadProgress.isDownloading && controlState.type === 'default') {
    setControlState({ type: 'progress' })
  }
  // Transition to progress when a new download starts
  if (downloadProgress.isDownloading && !prevDownloading.current) {
    if (controlState.type !== 'progress') {
      setControlState({ type: 'progress' })
    }
  }
  // Detect completion
  if (!downloadProgress.isDownloading && prevDownloading.current && controlState.type === 'progress') {
    setControlState({ type: 'completed' })
    completionTimerRef.current = setTimeout(() => {
      setControlState({ type: 'default' })
    }, 2500)
  }
  prevDownloading.current = downloadProgress.isDownloading

  // Cleanup timer
  useEffect(() => {
    return () => {
      if (completionTimerRef.current) clearTimeout(completionTimerRef.current)
    }
  }, [])

  // ---- Developer mode ----
  const developerMode = useDeveloperMode()

  // ---- Auto-search mutations ----
  const movieMutation = useAutoSearchMovie()
  const episodeMutation = useAutoSearchEpisode()
  const seasonMutation = useAutoSearchSeason()
  const seriesMutation = useAutoSearchSeries()
  const movieSlotMutation = useAutoSearchMovieSlot()
  const episodeSlotMutation = useAutoSearchEpisodeSlot()

  // ---- Handlers ----
  const handleManualSearch = useCallback(() => {
    setSearchModalOpen(true)
    setControlState({ type: 'searching', mode: 'manual' })
  }, [])

  const handleModalClose = useCallback((open: boolean) => {
    setSearchModalOpen(open)
    if (!open && controlState.type === 'searching' && controlState.mode === 'manual') {
      // Only revert if not already transitioned to progress
      if (!downloadProgress.isDownloading) {
        setControlState({ type: 'default' })
      }
    }
  }, [controlState, downloadProgress.isDownloading])

  const handleGrabSuccess = useCallback(() => {
    setControlState({ type: 'progress' })
  }, [])

  const handleAutoSearch = useCallback(async () => {
    setControlState({ type: 'searching', mode: 'auto' })
    try {
      if (developerMode) {
        await new Promise((r) => setTimeout(r, 5000))
      }
      switch (props.mediaType) {
        case 'movie': {
          const result = await movieMutation.mutateAsync(props.movieId)
          formatSingleResult(result, title)
          if (result.downloaded) {
            setControlState({ type: 'progress' })
          } else if (!result.found) {
            setControlState({ type: 'error', message: 'Not Found' })
          } else {
            setControlState({ type: 'default' })
          }
          break
        }
        case 'episode': {
          const result = await episodeMutation.mutateAsync(props.episodeId)
          formatSingleResult(result, title)
          if (result.downloaded) {
            setControlState({ type: 'progress' })
          } else if (!result.found) {
            setControlState({ type: 'error', message: 'Not Found' })
          } else {
            setControlState({ type: 'default' })
          }
          break
        }
        case 'season': {
          const result = await seasonMutation.mutateAsync({
            seriesId: props.seriesId,
            seasonNumber: props.seasonNumber,
          })
          formatBatchResult(result, `Season ${props.seasonNumber}`)
          if (result.downloaded > 0) {
            setControlState({ type: 'progress' })
          } else if (result.found === 0 && result.failed === 0) {
            setControlState({ type: 'error', message: 'Not Found' })
          } else {
            setControlState({ type: 'default' })
          }
          break
        }
        case 'series': {
          const result = await seriesMutation.mutateAsync(props.seriesId)
          formatBatchResult(result, title)
          if (result.downloaded > 0) {
            setControlState({ type: 'progress' })
          } else if (result.found === 0 && result.failed === 0) {
            setControlState({ type: 'error', message: 'Not Found' })
          } else {
            setControlState({ type: 'default' })
          }
          break
        }
        case 'movie-slot': {
          const result = await movieSlotMutation.mutateAsync({
            movieId: props.movieId,
            slotId: props.slotId,
          })
          if (result.downloaded) {
            toast.success(`Release grabbed for slot`)
            setControlState({ type: 'progress' })
          } else if (result.found) {
            toast.info('Release found but not grabbed')
            setControlState({ type: 'default' })
          } else {
            toast.warning('No releases found')
            setControlState({ type: 'error', message: 'Not Found' })
          }
          break
        }
        case 'episode-slot': {
          const result = await episodeSlotMutation.mutateAsync({
            episodeId: props.episodeId,
            slotId: props.slotId,
          })
          if (result.downloaded) {
            toast.success(`Release grabbed for slot`)
            setControlState({ type: 'progress' })
          } else if (result.found) {
            toast.info('Release found but not grabbed')
            setControlState({ type: 'default' })
          } else {
            toast.warning('No releases found')
            setControlState({ type: 'error', message: 'Not Found' })
          }
          break
        }
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`"${title}" is already in the download queue`)
        setControlState({ type: 'progress' })
      } else {
        toast.error(`Search failed for "${title}"`)
        setControlState({ type: 'error', message: 'Failed' })
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props, title, developerMode])

  const handleErrorDismiss = useCallback(() => {
    setControlState({ type: 'default' })
  }, [])

  const handleCompletionClick = useCallback(() => {
    if (completionTimerRef.current) {
      clearTimeout(completionTimerRef.current)
      completionTimerRef.current = null
    }
    setControlState({ type: 'default' })
  }, [])

  // ---- Search modal props ----
  const searchModalProps = buildSearchModalProps(props, qualityProfileId)

  // ---- Effective state: prefer progress if downloading regardless of controlState ----
  const effectiveState: ControlState = downloadProgress.isDownloading
    ? { type: 'progress' }
    : controlState

  // ---- Render ----
  const isDefault = effectiveState.type === 'default'
  const gapClass = size === 'lg' ? 'gap-2' : size === 'sm' ? 'gap-1.5' : 'gap-1'

  return (
    <div className={cn('relative', className)}>
      {/* Default buttons always rendered to maintain container width; invisible when non-default */}
      <div className={cn('flex items-center', gapClass, !isDefault && 'invisible')}>
        <DefaultButtons
          size={size}
          theme={theme}
          monitored={monitored}
          monitorDisabled={monitorDisabled}
          onManualSearch={handleManualSearch}
          onAutoSearch={handleAutoSearch}
          onMonitoredChange={onMonitoredChange}
        />
      </div>

      {/* Non-default states overlay at full width */}
      {!isDefault && (
        <div className="absolute inset-0 flex items-center">
          {effectiveState.type === 'searching' && (
            <SearchingState
              size={size}
              theme={theme}
              mode={effectiveState.mode}
            />
          )}

          {effectiveState.type === 'progress' && (
            <ProgressState
              size={size}
              theme={theme}
              progress={downloadProgress.progress}
              isPaused={downloadProgress.isPaused}
              releaseName={downloadProgress.releaseName}
              speed={downloadProgress.speed}
              eta={downloadProgress.eta}
              downloadedSize={downloadProgress.downloadedSize}
              totalSize={downloadProgress.size}
            />
          )}

          {effectiveState.type === 'completed' && (
            <CompletedState
              size={size}
              theme={theme}
              onClick={handleCompletionClick}
            />
          )}

          {effectiveState.type === 'error' && (
            <ErrorState
              size={size}
              theme={theme}
              message={effectiveState.message}
              onClick={handleErrorDismiss}
            />
          )}
        </div>
      )}

      <SearchModal
        open={searchModalOpen}
        onOpenChange={handleModalClose}
        onGrabSuccess={handleGrabSuccess}
        {...searchModalProps}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sub-components for each state
// ---------------------------------------------------------------------------

interface DefaultButtonsProps {
  size: ControlSize
  theme: MediaTheme
  monitored: boolean
  monitorDisabled?: boolean
  onManualSearch: () => void
  onAutoSearch: () => void
  onMonitoredChange: (monitored: boolean) => void
}

function DefaultButtons({
  size,
  theme,
  monitored,
  monitorDisabled,
  onManualSearch,
  onAutoSearch,
  onMonitoredChange,
}: DefaultButtonsProps) {
  if (size === 'lg') {
    return (
      <>
        <Button variant="outline" onClick={onManualSearch}>
          <UserSearch className="size-4 mr-2" />
          Search
        </Button>
        <Button variant="outline" onClick={onAutoSearch}>
          <Zap className="size-4 mr-2" />
          Auto Search
        </Button>
        <Button
          variant="outline"
          onClick={() => onMonitoredChange(!monitored)}
          disabled={monitorDisabled}
        >
          {monitored ? (
            <Eye className={cn('size-4 mr-2', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
          ) : (
            <EyeOff className="size-4 mr-2" />
          )}
          {monitored ? 'Monitored' : 'Unmonitored'}
        </Button>
      </>
    )
  }

  if (size === 'sm') {
    return (
      <>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button variant="outline" size="icon-sm" onClick={onManualSearch} />
            }
          >
            <UserSearch className="size-4" />
          </TooltipTrigger>
          <TooltipContent>Manual Search</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button variant="outline" size="icon-sm" onClick={onAutoSearch} />
            }
          >
            <Zap className="size-4" />
          </TooltipTrigger>
          <TooltipContent>Auto Search</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon-sm"
                onClick={() => onMonitoredChange(!monitored)}
                disabled={monitorDisabled}
              />
            }
          >
            {monitored ? (
              <Eye className={cn('size-4', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
            ) : (
              <EyeOff className="size-4" />
            )}
          </TooltipTrigger>
          <TooltipContent>{monitored ? 'Monitored' : 'Unmonitored'}</TooltipContent>
        </Tooltip>
      </>
    )
  }

  // xs
  return (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button variant="ghost" size="icon-sm" onClick={onManualSearch} />
          }
        >
          <UserSearch className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>Manual Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button variant="ghost" size="icon-sm" onClick={onAutoSearch} />
          }
        >
          <Zap className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>Auto Search</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => onMonitoredChange(!monitored)}
              disabled={monitorDisabled}
            />
          }
        >
          {monitored ? (
            <Eye className={cn('size-3.5', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
          ) : (
            <EyeOff className="size-3.5" />
          )}
        </TooltipTrigger>
        <TooltipContent>{monitored ? 'Monitored' : 'Unmonitored'}</TooltipContent>
      </Tooltip>
    </>
  )
}

interface SearchingStateProps {
  size: ControlSize
  theme: MediaTheme
  mode: 'manual' | 'auto'
}

function SearchingState({ size, theme, mode }: SearchingStateProps) {
  if (mode === 'manual') {
    return (
      <Button variant="outline" disabled className={cn('w-full', size === 'xs' ? 'h-6 text-xs' : size === 'sm' ? 'h-8 text-xs' : '')}>
        Searching...
      </Button>
    )
  }

  // Auto search
  if (size === 'xs') {
    const shimmerClass = theme === 'movie' ? 'shimmer-text-movie' : 'shimmer-text-tv'
    return (
      <Button variant="ghost" disabled className="w-full h-6 text-xs disabled:opacity-100">
        <span className={shimmerClass} data-text="Searching...">Searching...</span>
      </Button>
    )
  }

  const chasingClass = theme === 'movie' ? 'chasing-lights-movie' : 'chasing-lights-tv'

  if (size === 'lg') {
    return (
      <div className={cn(chasingClass, 'w-full')}>
        <div className="absolute inset-0 rounded-md bg-card z-[1]" />
        <Button variant="outline" disabled className="relative z-[2] w-full">
          Searching...
        </Button>
      </div>
    )
  }

  // sm
  return (
    <div className={cn(chasingClass, 'w-full')}>
      <div className="absolute inset-0 rounded-md bg-card z-[1]" />
      <Button variant="outline" disabled className="relative z-[2] w-full h-8 text-xs">
        Searching...
      </Button>
    </div>
  )
}

interface ProgressStateProps {
  size: ControlSize
  theme: MediaTheme
  progress: number
  isPaused: boolean
  releaseName: string
  speed: number
  eta: number
  downloadedSize: number
  totalSize: number
}

function ProgressState({
  size,
  theme,
  progress,
  isPaused,
  releaseName,
  speed,
  eta,
  downloadedSize,
  totalSize,
}: ProgressStateProps) {
  const progressContent = (
    <div
      className={cn(
        'relative overflow-hidden rounded-md w-full',
        isPaused && 'animation-paused',
        size === 'xs' ? 'h-6' : size === 'sm' ? 'h-8' : 'h-9',
      )}
    >
      {/* Background */}
      <div className="absolute inset-0 bg-muted/30" />

      {/* Progress fill */}
      <div
        className={cn(
          'absolute inset-y-0 left-0 transition-all duration-500 ease-out',
          theme === 'movie'
            ? 'bg-gradient-to-r from-movie-600/40 via-movie-500/50 to-movie-500/60'
            : 'bg-gradient-to-r from-tv-600/40 via-tv-500/50 to-tv-500/60',
        )}
        style={{ width: `${Math.max(progress, 2)}%` }}
      >
        {/* Shimmer */}
        {size !== 'xs' && (
          <div className="absolute inset-0 overflow-hidden">
            <div
              className={cn(
                'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
                theme === 'movie'
                  ? 'bg-gradient-to-r from-transparent via-movie-400/25 to-transparent'
                  : 'bg-gradient-to-r from-transparent via-tv-400/25 to-transparent',
              )}
            />
          </div>
        )}
      </div>

      {/* Edge glow */}
      {size !== 'xs' && (
        <div
          className={cn(
            'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
            theme === 'movie' ? 'bg-movie-400' : 'bg-tv-400',
          )}
          style={{ left: `calc(${Math.max(progress, 2)}% - 2px)` }}
        />
      )}

      {/* Inset glow */}
      {size !== 'xs' && (
        <div
          className={cn(
            'absolute inset-0 rounded-md ring-1 ring-inset',
            theme === 'movie'
              ? 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]'
              : 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
          )}
        />
      )}

      {/* Label */}
      <div className="absolute inset-0 flex items-center justify-center gap-2 text-sm text-muted-foreground">
        <Download className={size === 'xs' ? 'size-3.5' : 'size-4'} />
        {size === 'lg' && `Downloading${eta > 0 ? ` (${formatEta(eta)})` : ''}`}
      </div>
    </div>
  )

  const tooltipContent = (
    <div className="space-y-1 text-xs">
      {releaseName && <p className="font-medium max-w-64 truncate">{releaseName}</p>}
      <p>{progress.toFixed(1)}% — {formatBytes(downloadedSize)} / {formatBytes(totalSize)}</p>
      {!isPaused && speed > 0 && <p>{formatSpeed(speed)} — ETA: {formatEta(eta)}</p>}
      {isPaused && <p className="text-amber-400">Paused</p>}
    </div>
  )

  return (
    <Tooltip>
      <TooltipTrigger render={<div className="w-full" />}>
        {progressContent}
      </TooltipTrigger>
      <TooltipContent>{tooltipContent}</TooltipContent>
    </Tooltip>
  )
}

interface CompletedStateProps {
  size: ControlSize
  theme: MediaTheme
  onClick: () => void
}

function CompletedState({ size, theme, onClick }: CompletedStateProps) {
  const flashClass = theme === 'movie'
    ? 'animate-[download-complete-flash-movie_800ms_ease-out]'
    : 'animate-[download-complete-flash-tv_800ms_ease-out]'

  if (size === 'lg') {
    return (
      <Button
        variant="outline"
        className={cn(flashClass, 'cursor-pointer w-full')}
        onClick={onClick}
      >
        <Check className={cn('size-4 mr-2', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
        Downloaded
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="outline"
              size="icon-sm"
              className={cn(flashClass, 'cursor-pointer w-full')}
              onClick={onClick}
            />
          }
        >
          <Check className={cn('size-4', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
        </TooltipTrigger>
        <TooltipContent>Downloaded — click to dismiss</TooltipContent>
      </Tooltip>
    )
  }

  // xs
  return (
    <button onClick={onClick} className="p-1 w-full flex items-center justify-center">
      <Check className={cn('size-3.5', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
    </button>
  )
}

interface ErrorStateProps {
  size: ControlSize
  theme: MediaTheme
  message: string
  onClick: () => void
}

function ErrorState({ size, theme, message, onClick }: ErrorStateProps) {
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button variant="outline" className="cursor-pointer w-full" onClick={onClick}>
        <AlertCircle className={cn('size-4 mr-2', colorClass)} />
        {message}
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button variant="outline" size="icon-sm" className="cursor-pointer w-full" onClick={onClick} />
          }
        >
          <AlertCircle className={cn('size-4', colorClass)} />
        </TooltipTrigger>
        <TooltipContent>{message} — click to dismiss</TooltipContent>
      </Tooltip>
    )
  }

  // xs
  return (
    <Tooltip>
      <TooltipTrigger>
        <button onClick={onClick} className="p-1 w-full flex items-center justify-center">
          <AlertCircle className={cn('size-3.5', colorClass)} />
        </button>
      </TooltipTrigger>
      <TooltipContent>{message}</TooltipContent>
    </Tooltip>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildDownloadTarget(props: MediaSearchMonitorControlsProps): MediaTarget {
  switch (props.mediaType) {
    case 'movie':
      return { mediaType: 'movie', movieId: props.movieId }
    case 'series':
      return { mediaType: 'series', seriesId: props.seriesId }
    case 'season':
      return { mediaType: 'season', seriesId: props.seriesId, seasonNumber: props.seasonNumber }
    case 'episode':
      return { mediaType: 'episode', episodeId: props.episodeId, seriesId: props.seriesId, seasonNumber: props.seasonNumber }
    case 'movie-slot':
      return { mediaType: 'movie-slot', movieId: props.movieId, slotId: props.slotId }
    case 'episode-slot':
      return { mediaType: 'episode-slot', episodeId: props.episodeId, slotId: props.slotId, seriesId: props.seriesId, seasonNumber: props.seasonNumber }
  }
}

function buildSearchModalProps(
  props: MediaSearchMonitorControlsProps,
  qualityProfileId: number,
): Omit<React.ComponentProps<typeof SearchModal>, 'open' | 'onOpenChange' | 'onGrabSuccess'> {
  switch (props.mediaType) {
    case 'movie':
      return {
        qualityProfileId,
        movieId: props.movieId,
        movieTitle: props.title,
        tmdbId: props.tmdbId,
        imdbId: props.imdbId,
        year: props.year,
      }
    case 'series':
      return {
        qualityProfileId,
        seriesId: props.seriesId,
        seriesTitle: props.title,
        tvdbId: props.tvdbId,
      }
    case 'season':
      return {
        qualityProfileId,
        seriesId: props.seriesId,
        seriesTitle: props.seriesTitle,
        tvdbId: props.tvdbId,
        season: props.seasonNumber,
      }
    case 'episode':
      return {
        qualityProfileId,
        seriesId: props.seriesId,
        seriesTitle: props.seriesTitle,
        tvdbId: props.tvdbId,
        season: props.seasonNumber,
        episode: props.episodeNumber,
      }
    case 'movie-slot':
      return {
        qualityProfileId,
        movieId: props.movieId,
        movieTitle: props.title,
        tmdbId: props.tmdbId,
        imdbId: props.imdbId,
        year: props.year,
      }
    case 'episode-slot':
      return {
        qualityProfileId,
        seriesId: props.seriesId,
        seriesTitle: props.seriesTitle,
        tvdbId: props.tvdbId,
        season: props.seasonNumber,
        episode: props.episodeNumber,
      }
  }
}
