import { Loader2, Zap, Download } from 'lucide-react'
import { Button, buttonVariants } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  useAutoSearchMovie,
  useAutoSearchEpisode,
  useAutoSearchSeason,
  useAutoSearchSeries,
} from '@/hooks'
import { useDownloadingStore } from '@/stores'
import type { AutoSearchResult, BatchAutoSearchResult } from '@/types'
import type { VariantProps } from 'class-variance-authority'
import { toast } from 'sonner'

interface BaseAutoSearchButtonProps extends VariantProps<typeof buttonVariants> {
  title: string
  disabled?: boolean
  showLabel?: boolean
}

interface MovieAutoSearchButtonProps extends BaseAutoSearchButtonProps {
  mediaType: 'movie'
  movieId: number
}

interface EpisodeAutoSearchButtonProps extends BaseAutoSearchButtonProps {
  mediaType: 'episode'
  episodeId: number
}

interface SeasonAutoSearchButtonProps extends BaseAutoSearchButtonProps {
  mediaType: 'season'
  seriesId: number
  seasonNumber: number
}

interface SeriesAutoSearchButtonProps extends BaseAutoSearchButtonProps {
  mediaType: 'series'
  seriesId: number
}

export type AutoSearchButtonProps =
  | MovieAutoSearchButtonProps
  | EpisodeAutoSearchButtonProps
  | SeasonAutoSearchButtonProps
  | SeriesAutoSearchButtonProps

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

export function AutoSearchButton(props: AutoSearchButtonProps) {
  const { title, disabled, showLabel = true, variant = 'outline', size } = props

  const movieMutation = useAutoSearchMovie()
  const episodeMutation = useAutoSearchEpisode()
  const seasonMutation = useAutoSearchSeason()
  const seriesMutation = useAutoSearchSeries()

  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isMovieDownloading = (movieId: number) => {
    return queueItems.some(
      (item) =>
        item.movieId === movieId &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isSeriesDownloading = (seriesId: number) => {
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        item.isCompleteSeries &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isSeasonDownloading = (seriesId: number, seasonNumber: number) => {
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        ((item.seasonNumber === seasonNumber && item.isSeasonPack) ||
          item.isCompleteSeries) &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isEpisodeDownloading = (episodeId: number) => {
    return queueItems.some(
      (item) =>
        item.episodeId === episodeId &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const isPending =
    movieMutation.isPending ||
    episodeMutation.isPending ||
    seasonMutation.isPending ||
    seriesMutation.isPending

  // Check if the item is currently downloading
  const isDownloading = (() => {
    switch (props.mediaType) {
      case 'movie':
        return isMovieDownloading(props.movieId)
      case 'series':
        return isSeriesDownloading(props.seriesId)
      case 'season':
        return isSeasonDownloading(props.seriesId, props.seasonNumber)
      case 'episode':
        return isEpisodeDownloading(props.episodeId)
      default:
        return false
    }
  })()

  const handleClick = async () => {
    try {
      switch (props.mediaType) {
        case 'movie': {
          const result = await movieMutation.mutateAsync(props.movieId)
          formatSingleResult(result, title)
          break
        }
        case 'episode': {
          const result = await episodeMutation.mutateAsync(props.episodeId)
          formatSingleResult(result, title)
          break
        }
        case 'season': {
          const result = await seasonMutation.mutateAsync({
            seriesId: props.seriesId,
            seasonNumber: props.seasonNumber,
          })
          formatBatchResult(result, `Season ${props.seasonNumber}`)
          break
        }
        case 'series': {
          const result = await seriesMutation.mutateAsync(props.seriesId)
          formatBatchResult(result, title)
          break
        }
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`"${title}" is already in the download queue`)
      } else {
        toast.error(`Search failed for "${title}"`)
      }
    }
  }

  const buttonContent = (
    <>
      {isDownloading ? (
        <Download className="size-4 text-green-500" />
      ) : isPending ? (
        <Loader2 className="size-4 animate-spin" />
      ) : (
        <Zap className="size-4" />
      )}
      {showLabel && (isDownloading ? 'Downloading' : isPending ? 'Searching...' : 'Auto Search')}
    </>
  )

  if (!showLabel) {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant={variant}
              size={size === 'sm' ? 'icon-sm' : 'icon'}
              onClick={handleClick}
              disabled={disabled || isPending || isDownloading}
            />
          }
        >
          {buttonContent}
        </TooltipTrigger>
        <TooltipContent>
          <p>{isDownloading ? 'Downloading' : 'Auto Search'}</p>
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Button
      variant={variant}
      size={size}
      onClick={handleClick}
      disabled={disabled || isPending || isDownloading}
    >
      {buttonContent}
    </Button>
  )
}
