import { useMemo, useState } from 'react'

import { useNavigate, useParams } from '@tanstack/react-router'
import {
  Calendar,
  CalendarPlus,
  Clock,
  Drama,
  Edit,
  RefreshCw,
  SlidersVertical,
  Trash2,
  UserRoundPlus,
  UserStar,
} from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { BackdropImage } from '@/components/media/BackdropImage'
import { PosterImage } from '@/components/media/PosterImage'
import { ProductionStatusBadge } from '@/components/media/ProductionStatusBadge'
import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/RatingIcons'
import { StudioLogo } from '@/components/media/StudioLogo'
import { TitleTreatment } from '@/components/media/TitleTreatment'
import { MediaSearchMonitorControls } from '@/components/search'
import { SeasonList } from '@/components/series/SeasonList'
import { SeriesEditDialog } from '@/components/series/SeriesEditDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  useAssignEpisodeFile,
  useDeleteSeries,
  useEpisodes,
  useExtendedSeriesMetadata,
  useGlobalLoading,
  useMultiVersionSettings,
  useQualityProfiles,
  useRefreshSeries,
  useSeriesDetail,
  useSlots,
  useUpdateEpisodeMonitored,
  useUpdateSeasonMonitored,
  useUpdateSeries,
} from '@/hooks'
import { formatDate, formatRuntime, formatStatusSummary } from '@/lib/formatters'
import type { Episode } from '@/types'

export function SeriesDetailPage() {
  const { id } = useParams({ from: '/series/$id' })
  const navigate = useNavigate()
  const seriesId = Number.parseInt(id)

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  const globalLoading = useGlobalLoading()
  const { data: series, isLoading: queryLoading, isError, refetch } = useSeriesDetail(seriesId)
  const isLoading = queryLoading || globalLoading
  const { data: extendedData } = useExtendedSeriesMetadata(series?.tmdbId ?? 0)
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: episodes } = useEpisodes(seriesId)

  const updateMutation = useUpdateSeries()
  const deleteMutation = useDeleteSeries()
  const refreshMutation = useRefreshSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()
  const updateEpisodeMonitoredMutation = useUpdateEpisodeMonitored()

  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slots } = useSlots()
  const assignFileMutation = useAssignEpisodeFile()

  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = slots?.filter((s) => s.enabled) ?? []

  const extendedSeasons = extendedData?.seasons
  const episodeRatings = useMemo(() => {
    if (!extendedSeasons) {
      return undefined
    }
    const map: Record<number, Record<number, number>> = {}
    for (const season of extendedSeasons) {
      if (season.episodes) {
        const seasonMap: Record<number, number> = {}
        for (const ep of season.episodes) {
          if (ep.imdbRating) {
            seasonMap[ep.episodeNumber] = ep.imdbRating
          }
        }
        if (Object.keys(seasonMap).length > 0) {
          map[season.seasonNumber] = seasonMap
        }
      }
    }
    return Object.keys(map).length > 0 ? map : undefined
  }, [extendedSeasons])

  const handleAssignFileToSlot = async (fileId: number, episodeId: number, slotId: number) => {
    try {
      await assignFileMutation.mutateAsync({
        episodeId,
        slotId,
        data: { fileId },
      })
      refetch()
      toast.success('File assigned to slot')
    } catch {
      toast.error('Failed to assign file to slot')
    }
  }

  const handleToggleMonitored = async (newMonitored?: boolean) => {
    if (!series) {
      return
    }
    const target = newMonitored ?? !series.monitored
    try {
      await updateMutation.mutateAsync({
        id: series.id,
        data: { monitored: target },
      })
      toast.success(target ? 'Series monitored' : 'Series unmonitored')
    } catch {
      toast.error('Failed to update series')
    }
  }

  const handleRefresh = async () => {
    try {
      await refreshMutation.mutateAsync(seriesId)
      toast.success('Metadata refreshed')
    } catch {
      toast.error('Failed to refresh metadata')
    }
  }

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync({ id: seriesId })
      toast.success('Series deleted')
      navigate({ to: '/series' })
    } catch {
      toast.error('Failed to delete series')
    }
  }

  const handleSeasonMonitoredChange = async (seasonNumber: number, monitored: boolean) => {
    try {
      await updateSeasonMonitoredMutation.mutateAsync({
        seriesId,
        seasonNumber,
        monitored,
      })
      toast.success(`Season ${seasonNumber} ${monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error('Failed to update season')
    }
  }

  const handleEpisodeMonitoredChange = async (episode: Episode, monitored: boolean) => {
    try {
      await updateEpisodeMonitoredMutation.mutateAsync({
        seriesId,
        episodeId: episode.id,
        monitored,
      })
      toast.success(
        `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')} ${monitored ? 'monitored' : 'unmonitored'}`,
      )
    } catch {
      toast.error('Failed to update episode')
    }
  }

  if (isLoading) {
    return <LoadingState variant="detail" />
  }

  if (isError || !series) {
    return <ErrorState message="Series not found" onRetry={refetch} />
  }

  return (
    <div className="-m-6">
      {/* Hero with backdrop */}
      <div className="relative h-64 md:h-80">
        <BackdropImage
          tmdbId={series.tmdbId}
          tvdbId={series.tvdbId}
          type="series"
          alt={series.title}
          version={series.updatedAt}
          className="absolute inset-0"
        />
        {series.network ? (
          <StudioLogo
            tmdbId={series.tmdbId}
            type="series"
            alt={series.network}
            version={series.updatedAt}
            className="absolute top-4 right-4 z-10"
            fallback={
              <span className="rounded bg-black/50 px-2.5 py-1 text-xs font-medium text-white/80 backdrop-blur-sm">
                {series.network}
              </span>
            }
          />
        ) : null}
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex max-w-4xl items-end gap-6">
            {/* Poster */}
            <div className="hidden shrink-0 md:block">
              <PosterImage
                tmdbId={series.tmdbId}
                tvdbId={series.tvdbId}
                alt={series.title}
                type="series"
                version={series.updatedAt}
                className="h-60 w-40 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                <ProductionStatusBadge status={series.productionStatus} />
                <Badge variant="secondary">{formatStatusSummary(series.statusCounts)}</Badge>
                {qualityProfiles?.find((p) => p.id === series.qualityProfileId)?.name ? (
                  <Badge variant="secondary" className="gap-1">
                    <SlidersVertical className="size-3" />
                    {qualityProfiles.find((p) => p.id === series.qualityProfileId)?.name}
                  </Badge>
                ) : null}
              </div>
              <TitleTreatment
                tmdbId={series.tmdbId}
                tvdbId={series.tvdbId}
                type="series"
                alt={series.title}
                version={series.updatedAt}
                fallback={<h1 className="text-3xl font-bold text-white">{series.title}</h1>}
              />
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
                {extendedData?.contentRating ? (
                  <span className="shrink-0 rounded border border-gray-400 px-1.5 py-0.5 text-xs font-medium text-gray-300">
                    {extendedData.contentRating}
                  </span>
                ) : null}
                {series.year ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Calendar className="size-4 shrink-0" />
                    {series.year}
                  </span>
                ) : null}
                {series.runtime ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Clock className="size-4 shrink-0" />
                    {formatRuntime(series.runtime)}
                  </span>
                ) : null}
                {extendedData?.credits?.creators?.[0] ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserStar className="size-4 shrink-0" />
                    {extendedData.credits.creators[0].name}
                  </span>
                ) : null}
                {extendedData?.genres && extendedData.genres.length > 0 ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Drama className="size-4 shrink-0" />
                    {extendedData.genres.join(', ')}
                  </span>
                ) : null}
                {series.addedByUsername ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserRoundPlus className="size-4 shrink-0" />
                    {series.addedByUsername}
                  </span>
                ) : null}
                {series.addedAt ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <CalendarPlus className="size-4 shrink-0" />
                    {formatDate(series.addedAt)}
                  </span>
                ) : null}
              </div>
              {extendedData?.ratings &&
                (extendedData.ratings.rottenTomatoes !== undefined ||
                  extendedData.ratings.imdbRating !== undefined ||
                  extendedData.ratings.metacritic !== undefined) && (
                  <div className="flex items-center gap-4 text-sm text-gray-300">
                    {extendedData.ratings.rottenTomatoes !== undefined && (
                      <span className="flex items-center gap-1.5">
                        {extendedData.ratings.rottenTomatoes >= 60 ? (
                          <RTFreshIcon className="h-5" />
                        ) : (
                          <RTRottenIcon className="h-5" />
                        )}
                        <span className="font-medium">{extendedData.ratings.rottenTomatoes}%</span>
                      </span>
                    )}
                    {extendedData.ratings.imdbRating !== undefined && (
                      <span className="flex items-center gap-1.5">
                        <IMDbIcon className="h-4" />
                        <span className="font-medium">
                          {extendedData.ratings.imdbRating.toFixed(1)}
                        </span>
                      </span>
                    )}
                    {extendedData.ratings.metacritic !== undefined && (
                      <span className="flex items-center gap-1.5">
                        <MetacriticIcon className="h-5" />
                        <span className="font-medium">{extendedData.ratings.metacritic}</span>
                      </span>
                    )}
                  </div>
                )}
              {series.overview ? (
                <button
                  type="button"
                  className={`max-w-2xl cursor-pointer text-sm text-gray-300 text-left ${overviewExpanded ? '' : 'line-clamp-2'}`}
                  onClick={() => setOverviewExpanded(!overviewExpanded)}
                >
                  {series.overview}
                </button>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="bg-card flex flex-wrap gap-2 border-b px-6 py-4">
        <MediaSearchMonitorControls
          mediaType="series"
          seriesId={series.id}
          title={series.title}
          theme="tv"
          size="responsive"
          monitored={series.monitored}
          onMonitoredChange={handleToggleMonitored}
          qualityProfileId={series.qualityProfileId}
          tvdbId={series.tvdbId}
          tmdbId={series.tmdbId}
          imdbId={series.imdbId}
        />
        <div className="ml-auto flex gap-2">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="outline"
                  size="icon"
                  className="min-[820px]:hidden"
                  onClick={handleRefresh}
                  disabled={refreshMutation.isPending}
                />
              }
            >
              <RefreshCw className="size-4" />
            </TooltipTrigger>
            <TooltipContent>Refresh</TooltipContent>
          </Tooltip>
          <Button
            variant="outline"
            className="hidden min-[820px]:inline-flex"
            onClick={handleRefresh}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className="mr-2 size-4" />
            Refresh
          </Button>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="outline"
                  size="icon"
                  className="min-[820px]:hidden"
                  onClick={() => setEditDialogOpen(true)}
                />
              }
            >
              <Edit className="size-4" />
            </TooltipTrigger>
            <TooltipContent>Edit</TooltipContent>
          </Tooltip>
          <Button
            variant="outline"
            className="hidden min-[820px]:inline-flex"
            onClick={() => setEditDialogOpen(true)}
          >
            <Edit className="mr-2 size-4" />
            Edit
          </Button>
          <ConfirmDialog
            trigger={
              <>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button variant="destructive" size="icon" className="min-[820px]:hidden" />
                    }
                  >
                    <Trash2 className="size-4" />
                  </TooltipTrigger>
                  <TooltipContent>Delete</TooltipContent>
                </Tooltip>
                <Button variant="destructive" className="hidden min-[820px]:inline-flex">
                  <Trash2 className="mr-2 size-4" />
                  Delete
                </Button>
              </>
            }
            title="Delete series"
            description={`Are you sure you want to delete "${series.title}"? This action cannot be undone.`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={handleDelete}
          />
        </div>
      </div>

      {/* Content */}
      <div className="space-y-6 p-6">
        {/* Seasons */}
        <Card>
          <CardHeader>
            <CardTitle>Seasons & Episodes</CardTitle>
          </CardHeader>
          <CardContent>
            {series.seasons && series.seasons.length > 0 ? (
              <SeasonList
                seriesId={series.id}
                seriesTitle={series.title}
                qualityProfileId={series.qualityProfileId}
                tvdbId={series.tvdbId}
                tmdbId={series.tmdbId}
                imdbId={series.imdbId}
                seasons={series.seasons}
                episodes={episodes}
                onSeasonMonitoredChange={handleSeasonMonitoredChange}
                onEpisodeMonitoredChange={handleEpisodeMonitoredChange}
                onAssignFileToSlot={handleAssignFileToSlot}
                isMultiVersionEnabled={isMultiVersionEnabled}
                enabledSlots={enabledSlots}
                isAssigning={assignFileMutation.isPending}
                episodeRatings={episodeRatings}
              />
            ) : (
              <p className="text-muted-foreground">No seasons found</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Edit Dialog */}
      <SeriesEditDialog open={editDialogOpen} onOpenChange={setEditDialogOpen} series={series} />
    </div>
  )
}
