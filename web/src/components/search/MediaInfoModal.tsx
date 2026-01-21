import { useMemo } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Plus, Check, User, Download, Clock, CheckCircle } from 'lucide-react'
import { useExtendedMovieMetadata, useExtendedSeriesMetadata } from '@/hooks/useMetadata'
import { usePortalDownloads } from '@/hooks'
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Progress } from '@/components/ui/progress'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { PosterImage } from '@/components/media/PosterImage'
import { formatEta } from '@/lib/formatters'
import type {
  MovieSearchResult,
  SeriesSearchResult,
  ExtendedMovieResult,
  ExtendedSeriesResult,
  Person,
  ExternalRatings,
  SeasonResult,
  Request,
} from '@/types'

interface MediaInfoModalProps {
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
  actionIcon = <Plus className="size-4 mr-2" />,
  disabledLabel = 'Already in Library',
}: MediaInfoModalProps) {
  const navigate = useNavigate()
  const { data: downloads, requests } = usePortalDownloads()

  const movieQuery = useExtendedMovieMetadata(
    mediaType === 'movie' && open ? media.tmdbId : 0
  )
  const seriesQuery = useExtendedSeriesMetadata(
    mediaType === 'series' && open ? media.tmdbId : 0
  )

  const query = mediaType === 'movie' ? movieQuery : seriesQuery
  const extendedData = query.data as ExtendedMovieResult | ExtendedSeriesResult | undefined

  const tmdbId = media.tmdbId

  // Find ALL requests for this media (by TMDB ID)
  // For series, this includes both 'series' and 'season' type requests
  const matchingRequests = useMemo((): Request[] => {
    if (!requests || !tmdbId) return []
    return requests.filter((r) => {
      if (r.tmdbId !== tmdbId) return false
      if (mediaType === 'movie') return r.mediaType === 'movie'
      return r.mediaType === 'series' || r.mediaType === 'season'
    })
  }, [requests, tmdbId, mediaType])

  // Determine aggregate status across all requests
  const aggregateStatus = useMemo(() => {
    if (matchingRequests.length === 0) return null
    // If ALL requests are 'available', the item is fully in library
    if (matchingRequests.every((r) => r.status === 'available')) return 'available'
    // If ANY request is still pending, show pending
    if (matchingRequests.some((r) => r.status === 'pending')) return 'pending'
    // If ANY request is approved (but not available), show approved
    if (matchingRequests.some((r) => r.status === 'approved')) return 'approved'
    return matchingRequests[0].status
  }, [matchingRequests])

  // Find active download for this media
  const activeDownload = useMemo(() => {
    if (!downloads || !tmdbId) return undefined
    const requestIds = new Set(matchingRequests.map((r) => r.id))
    return downloads.find((d) => {
      if (d.tmdbId != null && d.tmdbId === tmdbId) return true
      if (requestIds.has(d.requestId)) return true
      return false
    })
  }, [downloads, tmdbId, matchingRequests])

  const hasActiveDownload = !!activeDownload
  const requestStatus = aggregateStatus
  const isApproved = requestStatus === 'approved'
  const isAvailable = requestStatus === 'available'
  const isPending = requestStatus === 'pending'
  const allRequestsHaveMediaId = matchingRequests.length > 0 && matchingRequests.every((r) => r.mediaId != null)
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
    if (!minutes) return null
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
      <DialogContent className="sm:max-w-2xl h-[85vh] flex flex-col overflow-hidden p-0">
        <DialogTitle className="sr-only">{media.title}</DialogTitle>
        <ScrollArea className="flex-1 min-h-0">
          <div className="p-4 space-y-4">
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
              <div className="flex-1 min-w-0 space-y-2">
                <h2 className="text-xl font-bold">
                  {media.title}
                  {media.year && (
                    <span className="text-muted-foreground font-normal ml-2">
                      ({media.year})
                    </span>
                  )}
                </h2>

                <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                  {extendedData?.contentRating && (
                    <Badge variant="outline">{extendedData.contentRating}</Badge>
                  )}
                  {media.runtime && <span>{formatRuntime(media.runtime)}</span>}
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
                  <div className="text-sm space-y-1">
                    {mediaType === 'movie' && director && (
                      <p>
                        <span className="text-muted-foreground">Director:</span>{' '}
                        {director}
                      </p>
                    )}
                    {mediaType === 'series' && creators && creators.length > 0 && (
                      <p>
                        <span className="text-muted-foreground">Created by:</span>{' '}
                        {creators.map((c) => c.name).join(', ')}
                      </p>
                    )}
                    {studio && (
                      <p>
                        <span className="text-muted-foreground">Studio:</span>{' '}
                        {studio}
                      </p>
                    )}
                  </div>
                )}

                {hasActiveDownload ? (
                  <div className="p-3 rounded-lg bg-purple-500/10 border border-purple-500/20 space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium text-purple-400 flex items-center gap-2">
                        <Download className="size-4" />
                        Downloading
                      </span>
                      <span className="text-muted-foreground text-xs">
                        {isPaused ? 'Paused' : isDownloading ? formatEta(eta) : 'Queued'}
                      </span>
                    </div>
                    <Progress value={progress} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground">
                      <span>{Math.round(progress)}%</span>
                      {isDownloading && downloadSpeed > 0 && (
                        <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
                      )}
                    </div>
                  </div>
                ) : isInLibrary || isAvailable ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Check className="size-4 mr-2" />
                    In Library
                  </Button>
                ) : isApproved ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Check className="size-4 mr-2" />
                    Approved
                  </Button>
                ) : isPending ? (
                  <Button variant="secondary" size="sm" disabled>
                    <Clock className="size-4 mr-2" />
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
            {media.overview && (
              <p className="text-sm text-muted-foreground leading-relaxed">
                {media.overview}
              </p>
            )}

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
                <h3 className="text-sm font-semibold mb-2">Cast</h3>
                <div className="flex gap-3 overflow-x-auto pb-2">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="flex flex-col items-center gap-1 shrink-0">
                      <Skeleton className="size-16 rounded-full" />
                      <Skeleton className="h-3 w-14" />
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              extendedData?.credits?.cast &&
              extendedData.credits.cast.length > 0 && (
                <CastList cast={extendedData.credits.cast} />
              )
            )}

            {/* Seasons (TV only) */}
            {mediaType === 'series' && seasons && seasons.length > 0 && (
              <SeasonsList seasons={seasons} />
            )}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function RatingsDisplay({ ratings }: { ratings: ExternalRatings }) {
  const hasRatings =
    ratings.rottenTomatoes ||
    ratings.rottenAudience ||
    ratings.imdbRating ||
    ratings.metacritic

  if (!hasRatings && !ratings.awards) return null

  return (
    <div className="space-y-2">
      {hasRatings && (
        <div className="flex flex-wrap items-center gap-4">
          {ratings.rottenTomatoes != null && (
            <div className="flex items-center gap-1.5">
              <span className="text-lg" role="img" aria-label="Rotten Tomatoes">
                {ratings.rottenTomatoes >= 60 ? '🍅' : '🟢'}
              </span>
              <span className="text-sm font-medium">{ratings.rottenTomatoes}%</span>
              <span className="text-xs text-muted-foreground">Critics</span>
            </div>
          )}
          {ratings.rottenAudience != null && (
            <div className="flex items-center gap-1.5">
              <span className="text-lg" role="img" aria-label="Audience Score">
                🍿
              </span>
              <span className="text-sm font-medium">{ratings.rottenAudience}%</span>
              <span className="text-xs text-muted-foreground">Audience</span>
            </div>
          )}
          {ratings.imdbRating != null && (
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-bold text-yellow-500">IMDb</span>
              <span className="text-sm font-medium">{ratings.imdbRating.toFixed(1)}</span>
              {ratings.imdbVotes != null && (
                <span className="text-xs text-muted-foreground">
                  ({ratings.imdbVotes.toLocaleString()} votes)
                </span>
              )}
            </div>
          )}
          {ratings.metacritic != null && (
            <div className="flex items-center gap-1.5">
              <span
                className={`text-xs font-bold px-1.5 py-0.5 rounded ${
                  ratings.metacritic >= 60
                    ? 'bg-green-600 text-white'
                    : ratings.metacritic >= 40
                      ? 'bg-yellow-500 text-black'
                      : 'bg-red-600 text-white'
                }`}
              >
                {ratings.metacritic}
              </span>
              <span className="text-xs text-muted-foreground">Metacritic</span>
            </div>
          )}
        </div>
      )}
      {ratings.awards && (
        <p className="text-sm text-muted-foreground">
          <span className="font-medium text-foreground">Awards:</span> {ratings.awards}
        </p>
      )}
    </div>
  )
}

function CastList({ cast }: { cast: Person[] }) {
  return (
    <div>
      <h3 className="text-sm font-semibold mb-2">Cast</h3>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {cast.slice(0, 12).map((person) => (
          <div
            key={person.id}
            className="flex flex-col items-center gap-1 shrink-0 w-20"
          >
            <div className="size-16 rounded-full bg-muted overflow-hidden flex items-center justify-center">
              {person.photoUrl ? (
                <img
                  src={person.photoUrl}
                  alt={person.name}
                  className="size-full object-cover"
                />
              ) : (
                <User className="size-8 text-muted-foreground" />
              )}
            </div>
            <span className="text-xs text-center line-clamp-2 w-full">{person.name}</span>
            {person.role && (
              <span className="text-xs text-muted-foreground text-center line-clamp-2 w-full">
                {person.role}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

function SeasonsList({ seasons }: { seasons: SeasonResult[] }) {
  const regularSeasons = seasons.filter((s) => s.seasonNumber > 0)

  if (regularSeasons.length === 0) return null

  const isFutureDate = (dateStr?: string) => {
    if (!dateStr) return false
    const date = new Date(dateStr)
    return date > new Date()
  }

  const isSeasonFuture = (season: SeasonResult) => {
    const firstEpisode = season.episodes?.[0]
    return firstEpisode ? isFutureDate(firstEpisode.airDate) : false
  }

  return (
    <div>
      <h3 className="text-sm font-semibold mb-2">
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
                  {season.episodes && (
                    <Badge variant="secondary" className="text-xs">
                      {season.episodes.length} episodes
                    </Badge>
                  )}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                {season.overview && (
                  <p className="text-sm text-muted-foreground mb-2">{season.overview}</p>
                )}
                {season.episodes && season.episodes.length > 0 && (
                  <div className="space-y-1">
                    {season.episodes.map((ep) => {
                      const isFuture = isFutureDate(ep.airDate)
                      return (
                        <div
                          key={ep.episodeNumber}
                          className={`text-sm flex items-baseline gap-2 ${isFuture ? 'text-muted-foreground' : ''}`}
                        >
                          <span className={isFuture ? '' : 'text-muted-foreground'}>
                            E{ep.episodeNumber}
                          </span>
                          <span className="truncate">{ep.title}</span>
                          {ep.airDate && (
                            <span className="text-xs text-muted-foreground ml-auto shrink-0">
                              {ep.airDate}
                            </span>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )}
              </AccordionContent>
            </AccordionItem>
          )
        })}
      </Accordion>
    </div>
  )
}
