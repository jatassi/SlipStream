import { useState, useMemo } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import {
  UserSearch,
  RefreshCw,
  Trash2,
  Edit,
  Calendar,
  CalendarPlus,
  Clock,
  UserStar,
  UserRoundPlus,
  Eye,
  EyeOff,
  SlidersVertical,
  Drama,
} from 'lucide-react'
import { BackdropImage } from '@/components/media/BackdropImage'
import { PosterImage } from '@/components/media/PosterImage'
import { TitleTreatment } from '@/components/media/TitleTreatment'
import { StudioLogo } from '@/components/media/StudioLogo'
import { RTFreshIcon, RTRottenIcon, IMDbIcon, MetacriticIcon } from '@/components/media/RatingIcons'
import { ProductionStatusBadge } from '@/components/media/ProductionStatusBadge'
import { formatStatusSummary } from '@/lib/formatters'
import { SeasonList } from '@/components/series/SeasonList'
import { SeriesEditDialog } from '@/components/series/SeriesEditDialog'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { SearchModal } from '@/components/search/SearchModal'
import { AutoSearchButton } from '@/components/search/AutoSearchButton'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  useSeriesDetail,
  useUpdateSeries,
  useDeleteSeries,
  useRefreshSeries,
  useEpisodes,
  useUpdateSeasonMonitored,
  useUpdateEpisodeMonitored,
  useAutoSearchEpisodeSlot,
  useMultiVersionSettings,
  useSlots,
  useAssignEpisodeFile,
  useQualityProfiles,
  useExtendedSeriesMetadata,
} from '@/hooks'
import { formatRuntime, formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
import type { Episode } from '@/types'

interface SearchContext {
  season?: number
  episode?: Episode
  qualityProfileId?: number | null
}

export function SeriesDetailPage() {
  const { id } = useParams({ from: '/series/$id' })
  const navigate = useNavigate()
  const seriesId = parseInt(id)

  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [searchContext, setSearchContext] = useState<SearchContext>({})
  const [searchingSlotId, setSearchingSlotId] = useState<number | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  const { data: series, isLoading, isError, refetch } = useSeriesDetail(seriesId)
  const { data: extendedData } = useExtendedSeriesMetadata(series?.tmdbId ?? 0)
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: episodes } = useEpisodes(seriesId)

  const updateMutation = useUpdateSeries()
  const deleteMutation = useDeleteSeries()
  const refreshMutation = useRefreshSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()
  const updateEpisodeMonitoredMutation = useUpdateEpisodeMonitored()
  const episodeSlotAutoSearchMutation = useAutoSearchEpisodeSlot()

  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slots } = useSlots()
  const assignFileMutation = useAssignEpisodeFile()

  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = slots?.filter(s => s.enabled) ?? []

  const episodeRatings = useMemo(() => {
    if (!extendedData?.seasons) return undefined
    const map: Record<number, Record<number, number>> = {}
    for (const season of extendedData.seasons) {
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
  }, [extendedData?.seasons])

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

  const handleToggleMonitored = async () => {
    if (!series) return
    try {
      await updateMutation.mutateAsync({
        id: series.id,
        data: { monitored: !series.monitored },
      })
      toast.success(series.monitored ? 'Series unmonitored' : 'Series monitored')
    } catch {
      toast.error('Failed to update series')
    }
  }

  const handleManualSearch = () => {
    setSearchContext({})
    setSearchModalOpen(true)
  }

  const handleSeasonSearch = (seasonNumber: number) => {
    setSearchContext({ season: seasonNumber })
    setSearchModalOpen(true)
  }

  const handleEpisodeSearch = (episode: Episode) => {
    setSearchContext({ season: episode.seasonNumber, episode })
    setSearchModalOpen(true)
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
      toast.success(`S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')} ${monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error('Failed to update episode')
    }
  }

  const handleSlotManualSearch = (episodeId: number, slotId: number) => {
    const slot = slots?.find(s => s.id === slotId)
    const episode = episodes?.find(e => e.id === episodeId)
    if (!slot?.qualityProfileId) {
      toast.error('Slot has no quality profile configured')
      return
    }
    if (episode) {
      setSearchContext({
        season: episode.seasonNumber,
        episode,
        qualityProfileId: slot.qualityProfileId,
      })
      setSearchModalOpen(true)
    }
  }

  const handleSlotAutoSearch = async (episodeId: number, slotId: number) => {
    const slot = slots?.find(s => s.id === slotId)
    const episode = episodes?.find(e => e.id === episodeId)
    if (!slot?.qualityProfileId) {
      toast.error('Slot has no quality profile configured')
      return
    }

    setSearchingSlotId(slotId)
    try {
      const result = await episodeSlotAutoSearchMutation.mutateAsync({ episodeId, slotId })
      const epLabel = episode
        ? `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`
        : 'Episode'
      if (result.downloaded) {
        toast.success(`Release grabbed for ${slot.name} (${epLabel})`)
        refetch()
      } else if (result.found) {
        toast.info(`Release found for ${slot.name} (${epLabel}) but not grabbed`)
      } else {
        toast.info(`No releases found for ${slot.name} (${epLabel})`)
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning('Episode is already in the download queue')
      } else {
        toast.error(`Auto search failed for ${slot.name}`)
      }
    } finally {
      setSearchingSlotId(null)
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
        {series.network && (
          <StudioLogo
            tmdbId={series.tmdbId}
            type="series"
            alt={series.network}
            version={series.updatedAt}
            className="absolute top-4 right-4 z-10"
            fallback={
              <span className="px-2.5 py-1 rounded bg-black/50 text-xs font-medium text-white/80 backdrop-blur-sm">
                {series.network}
              </span>
            }
          />
        )}
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex gap-6 items-end max-w-4xl">
            {/* Poster */}
            <div className="hidden md:block shrink-0">
              <PosterImage
                tmdbId={series.tmdbId}
                tvdbId={series.tvdbId}
                alt={series.title}
                type="series"
                version={series.updatedAt}
                className="w-40 h-60 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex items-center gap-2 flex-wrap">
                <ProductionStatusBadge status={series.productionStatus} />
                <Badge variant="secondary">
                  {formatStatusSummary(series.statusCounts)}
                </Badge>
                {qualityProfiles?.find((p) => p.id === series.qualityProfileId)?.name && (
                  <Badge variant="secondary" className="gap-1">
                    <SlidersVertical className="size-3" />
                    {qualityProfiles.find((p) => p.id === series.qualityProfileId)?.name}
                  </Badge>
                )}
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
                {extendedData?.contentRating && (
                  <span className="shrink-0 px-1.5 py-0.5 border border-gray-400 rounded text-xs font-medium text-gray-300">
                    {extendedData.contentRating}
                  </span>
                )}
                {series.year && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Calendar className="size-4 shrink-0" />
                    {series.year}
                  </span>
                )}
                {series.runtime && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Clock className="size-4 shrink-0" />
                    {formatRuntime(series.runtime)}
                  </span>
                )}
                {extendedData?.credits?.creators?.[0] && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserStar className="size-4 shrink-0" />
                    {extendedData.credits.creators[0].name}
                  </span>
                )}
                {extendedData?.genres && extendedData.genres.length > 0 && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Drama className="size-4 shrink-0" />
                    {extendedData.genres.join(', ')}
                  </span>
                )}
                {series.addedByUsername && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserRoundPlus className="size-4 shrink-0" />
                    {series.addedByUsername}
                  </span>
                )}
                {series.addedAt && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <CalendarPlus className="size-4 shrink-0" />
                    {formatDate(series.addedAt)}
                  </span>
                )}
              </div>
              {(extendedData?.ratings?.rottenTomatoes != null || extendedData?.ratings?.imdbRating != null || extendedData?.ratings?.metacritic != null) && (
                <div className="flex items-center gap-4 text-sm text-gray-300">
                  {extendedData?.ratings?.rottenTomatoes != null && (
                    <span className="flex items-center gap-1.5">
                      {extendedData.ratings.rottenTomatoes >= 60 ? (
                        <RTFreshIcon className="h-5" />
                      ) : (
                        <RTRottenIcon className="h-5" />
                      )}
                      <span className="font-medium">{extendedData.ratings.rottenTomatoes}%</span>
                    </span>
                  )}
                  {extendedData?.ratings?.imdbRating != null && (
                    <span className="flex items-center gap-1.5">
                      <IMDbIcon className="h-4" />
                      <span className="font-medium">{extendedData.ratings.imdbRating.toFixed(1)}</span>
                    </span>
                  )}
                  {extendedData?.ratings?.metacritic != null && (
                    <span className="flex items-center gap-1.5">
                      <MetacriticIcon className="h-5" />
                      <span className="font-medium">{extendedData.ratings.metacritic}</span>
                    </span>
                  )}
                </div>
              )}
              {series.overview && (
                <p
                  className={`text-sm text-gray-300 max-w-2xl cursor-pointer ${overviewExpanded ? '' : 'line-clamp-2'}`}
                  onClick={() => setOverviewExpanded(!overviewExpanded)}
                >
                  {series.overview}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="px-6 py-4 border-b bg-card flex flex-wrap gap-2">
        <Button variant="outline" onClick={handleManualSearch}>
          <UserSearch className="size-4 mr-2" />
          Search
        </Button>
        <AutoSearchButton
          mediaType="series"
          seriesId={series.id}
          title={series.title}
        />
        <Button variant="outline" onClick={handleToggleMonitored} className={series.monitored ? 'glow-tv-sm' : ''}>
          {series.monitored ? (
            <>
              <Eye className="size-4 mr-2 text-tv-400" />
              Monitored
            </>
          ) : (
            <>
              <EyeOff className="size-4 mr-2" />
              Unmonitored
            </>
          )}
        </Button>
        <div className="ml-auto flex gap-2">
          <Button
            variant="outline"
            onClick={handleRefresh}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className="size-4 mr-2" />
            Refresh
          </Button>
          <Button variant="outline" onClick={() => setEditDialogOpen(true)}>
            <Edit className="size-4 mr-2" />
            Edit
          </Button>
          <ConfirmDialog
            trigger={
              <Button variant="destructive">
                <Trash2 className="size-4 mr-2" />
                Delete
              </Button>
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
      <div className="p-6 space-y-6">
        {/* Seasons */}
        <Card>
          <CardHeader>
            <CardTitle>Seasons & Episodes</CardTitle>
          </CardHeader>
          <CardContent>
            {series.seasons && series.seasons.length > 0 ? (
              <SeasonList
                seriesId={series.id}
                seasons={series.seasons}
                episodes={episodes}
                onSeasonMonitoredChange={handleSeasonMonitoredChange}
                onSeasonSearch={handleSeasonSearch}
                onEpisodeSearch={handleEpisodeSearch}
                onEpisodeMonitoredChange={handleEpisodeMonitoredChange}
                onAssignFileToSlot={handleAssignFileToSlot}
                onSlotManualSearch={handleSlotManualSearch}
                onSlotAutoSearch={handleSlotAutoSearch}
                searchingSlotId={searchingSlotId}
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

      {/* Search Modal */}
      <SearchModal
        open={searchModalOpen}
        onOpenChange={(open) => {
          setSearchModalOpen(open)
          if (!open) setSearchContext({})
        }}
        qualityProfileId={searchContext.qualityProfileId ?? series.qualityProfileId}
        seriesId={series.id}
        seriesTitle={series.title}
        tvdbId={series.tvdbId}
        season={searchContext.season}
        episode={searchContext.episode?.episodeNumber}
      />

      {/* Edit Dialog */}
      <SeriesEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        series={series}
      />
    </div>
  )
}
