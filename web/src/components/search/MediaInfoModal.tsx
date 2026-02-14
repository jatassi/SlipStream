import { useMemo } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Check, Clock, Download, Plus, User } from 'lucide-react'

import { PosterImage } from '@/components/media/PosterImage'
import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/RatingIcons'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { usePortalDownloads } from '@/hooks'
import { useExtendedMovieMetadata, useExtendedSeriesMetadata } from '@/hooks/useMetadata'
import { formatEta } from '@/lib/formatters'
import type {
  ExtendedMovieResult,
  ExtendedSeriesResult,
  ExternalRatings,
  MovieSearchResult,
  Person,
  Request,
  SeasonResult,
  SeriesSearchResult,
} from '@/types'

type MediaInfoModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  media: MovieSearchResult | SeriesSearchResult
  mediaType: 'movie' | 'series'
  inLibrary?: boolean
  onAction?: () => void
  actionLabel?: string
  actionIcon?: React.ReactNode
  disabledLabel?: string
}

export function MediaInfoModal({
  open,
  onOpenChange,
  media,
  mediaType,
  inLibrary,
  onAction,
  actionLabel = 'Add to Library',
  actionIcon = <Plus className="mr-2 size-4" />,
}: MediaInfoModalProps) {
  const navigate = useNavigate()
  const { data: downloads, requests } = usePortalDownloads()

  const movieQuery = useExtendedMovieMetadata(mediaType === 'movie' && open ? media.tmdbId : 0)
  const seriesQuery = useExtendedSeriesMetadata(mediaType === 'series' && open ? media.tmdbId : 0)

  const query = mediaType === 'movie' ? movieQuery : seriesQuery
  const extendedData = query.data

  const tmdbId = media.tmdbId

  // Find ALL requests for this media (by TMDB ID)
  // For series, this includes both 'series' and 'season' type requests
  const matchingRequests = useMemo((): Request[] => {
    if (!requests || !tmdbId) {
      return []
    }
    return requests.filter((r) => {
      if (r.tmdbId !== tmdbId) {
        return false
      }
      if (mediaType === 'movie') {
        return r.mediaType === 'movie'
      }
      return r.mediaType === 'series' || r.mediaType === 'season'
    })
  }, [requests, tmdbId, mediaType])

  // Determine aggregate status across all requests
  const aggregateStatus = useMemo(() => {
    if (matchingRequests.length === 0) {
      return null
    }
    // If ALL requests are 'available', the item is fully in library
    if (matchingRequests.every((r) => r.status === 'available')) {
      return 'available'
    }
    // If ANY request is still pending, show pending
    if (matchingRequests.some((r) => r.status === 'pending')) {
      return 'pending'
    }
    // If ANY request is approved (but not available), show approved
    if (matchingRequests.some((r) => r.status === 'approved')) {
      return 'approved'
    }
    return matchingRequests[0].status
  }, [matchingRequests])

  // Find active download for this media
  const activeDownload = useMemo(() => {
    if (!downloads || !tmdbId) {
      return undefined
    }
    const requestIds = new Set(matchingRequests.map((r) => r.id))
    return downloads.find((d) => {
      if (d.tmdbId != null && d.tmdbId === tmdbId) {
        return true
      }
      if (requestIds.has(d.requestId)) {
        return true
      }
      return false
    })
  }, [downloads, tmdbId, matchingRequests])

  const hasActiveDownload = !!activeDownload
  const requestStatus = aggregateStatus
  const isApproved = requestStatus === 'approved'
  const isAvailable = requestStatus === 'available'
  const isPending = requestStatus === 'pending'
  const allRequestsHaveMediaId =
    matchingRequests.length > 0 && matchingRequests.every((r) => r.mediaId != null)
  const isInLibrary = inLibrary || (isAvailable && allRequestsHaveMediaId)

  // Download progress stats
  const progress = activeDownload ? activeDownload.progress : 0
  const downloadSpeed = activeDownload?.downloadSpeed ?? 0
  const eta = activeDownload?.eta ?? 0
  const isDownloading = activeDownload?.status === 'downloading'
  const isPaused = activeDownload?.status === 'paused'

  const handleAdd = () => {
    if (onAction) {
      onAction()
      onOpenChange(false)
      return
    }
    if (mediaType === 'movie') {
      navigate({ to: '/movies/add', search: { tmdbId: media.tmdbId } })
    } else {
      navigate({ to: '/series/add', search: { tmdbId: media.tmdbId } })
    }
    onOpenChange(false)
  }

  const formatRuntime = (minutes?: number) => {
    if (!minutes) {
      return null
    }
    const hours = Math.floor(minutes / 60)
    const mins = minutes % 60
    return hours > 0 ? `${hours}h ${mins}m` : `${mins}m`
  }

  const director = extendedData?.credits?.directors?.[0]?.name
  const creators = (extendedData as ExtendedSeriesResult)?.credits?.creators
  const studio = (extendedData as ExtendedMovieResult)?.studio
  const seasons = (extendedData as ExtendedSeriesResult)?.seasons

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[85vh] flex-col overflow-hidden p-0 sm:max-w-2xl">
        <DialogTitle className="sr-only">{media.title}</DialogTitle>
        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-4 p-4">
            {/* Header with poster and basic info */}
            <div className="flex gap-4">
              <div className="w-28 shrink-0">
                <PosterImage
                  url={media.posterUrl}
                  alt={media.title}
                  type={mediaType}
                  className="rounded-lg"
                />
              </div>
              <div className="min-w-0 flex-1 space-y-2">
                <h2 className="text-xl font-bold">
                  {media.title}
                  {media.year ? (
                    <span className="text-muted-foreground ml-2 font-normal">({media.year})</span>
                  ) : null}
                </h2>

                <div className="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
                  {extendedData?.contentRating ? (
                    <Badge variant="outline">{extendedData.contentRating}</Badge>
                  ) : null}
                  {media.runtime ? <span>{formatRuntime(media.runtime)}</span> : null}
                  {media.genres?.slice(0, 3).map((genre) => (
                    <Badge key={genre} variant="secondary">
                      {genre}
                    </Badge>
                  ))}
                </div>

                {query.isLoading ? (
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-4 w-24" />
                  </div>
                ) : (
                  <div className="space-y-1 text-sm">
                    {mediaType === 'movie' && director ? (
                      <p>
                        <span className="text-muted-foreground">Director:</span> {director}
                      </p>
                    ) : null}
                    {mediaType === 'series' && creators && creators.length > 0 ? (
                      <p>
                        <span className="text-muted-foreground">Created by:</span>{' '}
                        {creators.map((c) => c.name).join(', ')}
                      </p>
                    ) : null}
                    {studio ? (
                      <p>
                        <span className="text-muted-foreground">Studio:</span> {studio}
                      </p>
                    ) : null}
                  </div>
                )}

                {hasActiveDownload ? (
                  <div className="space-y-2 rounded-lg border border-purple-500/20 bg-purple-500/10 p-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2 font-medium text-purple-400">
                        <Download className="size-4" />
                        Downloading
                      </span>
                      <span className="text-muted-foreground text-xs">
                        {isPaused ? 'Paused' : isDownloading ? formatEta(eta) : 'Queued'}
                      </span>
                    </div>
                    <Progress value={progress} className="h-2" />
                    <div className="text-muted-foreground flex justify-between text-xs">
                      <span>{Math.round(progress)}%</span>
                      {isDownloading && downloadSpeed > 0 ? (
                        <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
                      ) : null}
                    </div>
                  </div>
                ) : isInLibrary || isAvailable ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Check className="mr-2 size-4" />
                    In Library
                  </Button>
                ) : isApproved ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Check className="mr-2 size-4" />
                    Approved
                  </Button>
                ) : isPending ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Clock className="mr-2 size-4" />
                    Requested
                  </Button>
                ) : onAction ? (
                  <Button variant="default" size="sm" onClick={handleAdd}>
                    {actionIcon}
                    {actionLabel}
                  </Button>
                ) : null}
              </div>
            </div>

            {/* Overview */}
            {media.overview ? (
              <p className="text-muted-foreground text-sm leading-relaxed">{media.overview}</p>
            ) : null}

            {/* Ratings */}
            {query.isLoading ? (
              <div className="flex gap-4">
                <Skeleton className="h-8 w-20" />
                <Skeleton className="h-8 w-20" />
                <Skeleton className="h-8 w-20" />
              </div>
            ) : (
              extendedData?.ratings && <RatingsDisplay ratings={extendedData.ratings} />
            )}

            {/* Cast */}
            {query.isLoading ? (
              <div>
                <h3 className="mb-2 text-sm font-semibold">Cast</h3>
                <div className="flex gap-3 overflow-x-auto pb-2">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="flex shrink-0 flex-col items-center gap-1">
                      <Skeleton className="size-16 rounded-full" />
                      <Skeleton className="h-3 w-14" />
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              extendedData?.credits?.cast &&
              extendedData.credits.cast.length > 0 && <CastList cast={extendedData.credits.cast} />
            )}

            {/* Seasons (TV only) */}
            {mediaType === 'series' && seasons && seasons.length > 0 ? (
              <SeasonsList seasons={seasons} />
            ) : null}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function RatingsDisplay({ ratings }: { ratings: ExternalRatings }) {
  const hasRatings =
    ratings.rottenTomatoes || ratings.rottenAudience || ratings.imdbRating || ratings.metacritic

  if (!hasRatings && !ratings.awards) {
    return null
  }

  return (
    <div className="space-y-2">
      {hasRatings ? (
        <div className="flex flex-wrap items-center gap-4">
          {ratings.rottenTomatoes != null && (
            <div className="flex items-center gap-1.5">
              {ratings.rottenTomatoes >= 60 ? (
                <RTFreshIcon className="h-5" />
              ) : (
                <RTRottenIcon className="h-5" />
              )}
              <span className="text-sm font-medium">{ratings.rottenTomatoes}%</span>
              <span className="text-muted-foreground text-xs">Critics</span>
            </div>
          )}
          {ratings.rottenAudience != null && (
            <div className="flex items-center gap-1.5">
              {ratings.rottenAudience >= 60 ? (
                <RTFreshIcon className="h-5" />
              ) : (
                <RTRottenIcon className="h-5" />
              )}
              <span className="text-sm font-medium">{ratings.rottenAudience}%</span>
              <span className="text-muted-foreground text-xs">Audience</span>
            </div>
          )}
          {ratings.imdbRating != null && (
            <div className="flex items-center gap-1.5">
              <IMDbIcon className="h-4" />
              <span className="text-sm font-medium">{ratings.imdbRating.toFixed(1)}</span>
              {ratings.imdbVotes != null && (
                <span className="text-muted-foreground text-xs">
                  ({ratings.imdbVotes.toLocaleString()} votes)
                </span>
              )}
            </div>
          )}
          {ratings.metacritic != null && (
            <div className="flex items-center gap-1.5">
              <MetacriticIcon className="h-5" />
              <span className="text-sm font-medium">{ratings.metacritic}</span>
            </div>
          )}
        </div>
      ) : null}
      {ratings.awards ? (
        <p className="text-muted-foreground text-sm">
          <span className="text-foreground font-medium">Awards:</span> {ratings.awards}
        </p>
      ) : null}
    </div>
  )
}

function CastList({ cast }: { cast: Person[] }) {
  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">Cast</h3>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {cast.slice(0, 12).map((person) => (
          <div key={person.id} className="flex w-20 shrink-0 flex-col items-center gap-1">
            <div className="bg-muted flex size-16 items-center justify-center overflow-hidden rounded-full">
              {person.photoUrl ? (
                <img src={person.photoUrl} alt={person.name} className="size-full object-cover" />
              ) : (
                <User className="text-muted-foreground size-8" />
              )}
            </div>
            <span className="line-clamp-2 w-full text-center text-xs">{person.name}</span>
            {person.role ? (
              <span className="text-muted-foreground line-clamp-2 w-full text-center text-xs">
                {person.role}
              </span>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  )
}

function SeasonsList({ seasons }: { seasons: SeasonResult[] }) {
  const regularSeasons = seasons.filter((s) => s.seasonNumber > 0)

  if (regularSeasons.length === 0) {
    return null
  }

  const isFutureDate = (dateStr?: string) => {
    if (!dateStr) {
      return false
    }
    const date = new Date(dateStr)
    return date > new Date()
  }

  const isSeasonFuture = (season: SeasonResult) => {
    const firstEpisode = season.episodes?.[0]
    return firstEpisode ? isFutureDate(firstEpisode.airDate) : false
  }

  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">
        {regularSeasons.length} {regularSeasons.length === 1 ? 'Season' : 'Seasons'}
      </h3>
      <Accordion>
        {regularSeasons.map((season) => {
          const seasonIsFuture = isSeasonFuture(season)
          return (
            <AccordionItem key={season.seasonNumber} value={`season-${season.seasonNumber}`}>
              <AccordionTrigger>
                <div className="flex items-center gap-2">
                  <span className={seasonIsFuture ? 'text-muted-foreground' : ''}>
                    {season.name || `Season ${season.seasonNumber}`}
                  </span>
                  {season.episodes ? (
                    <Badge variant="secondary" className="text-xs">
                      {season.episodes.length} episodes
                    </Badge>
                  ) : null}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                {season.overview ? (
                  <p className="text-muted-foreground mb-2 text-sm">{season.overview}</p>
                ) : null}
                {season.episodes && season.episodes.length > 0 ? (
                  <div className="space-y-1">
                    {season.episodes.map((ep) => {
                      const isFuture = isFutureDate(ep.airDate)
                      return (
                        <div
                          key={ep.episodeNumber}
                          className={`flex items-baseline gap-2 text-sm ${isFuture ? 'text-muted-foreground' : ''}`}
                        >
                          <span className={isFuture ? '' : 'text-muted-foreground'}>
                            E{ep.episodeNumber}
                          </span>
                          <span className="truncate">{ep.title}</span>
                          {ep.airDate ? (
                            <span className="text-muted-foreground ml-auto shrink-0 text-xs">
                              {ep.airDate}
                            </span>
                          ) : null}
                        </div>
                      )
                    })}
                  </div>
                ) : null}
              </AccordionContent>
            </AccordionItem>
          )
        })}
      </Accordion>
    </div>
  )
}
