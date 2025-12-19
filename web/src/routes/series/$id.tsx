import { useParams, useNavigate } from '@tanstack/react-router'
import {
  Search,
  RefreshCw,
  Trash2,
  Edit,
  Calendar,
  Clock,
  HardDrive,
  Bookmark,
  BookmarkX,
  Tv,
} from 'lucide-react'
import { BackdropImage } from '@/components/media/BackdropImage'
import { PosterImage } from '@/components/media/PosterImage'
import { StatusBadge } from '@/components/media/StatusBadge'
import { SeasonList } from '@/components/series/SeasonList'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  useSeriesDetail,
  useUpdateSeries,
  useDeleteSeries,
  useSearchSeries,
  useRefreshSeries,
  useEpisodes,
  useUpdateSeasonMonitored,
} from '@/hooks'
import { formatBytes, formatRuntime, formatDate } from '@/lib/formatters'
import { toast } from 'sonner'

export function SeriesDetailPage() {
  const { id } = useParams({ from: '/series/$id' })
  const navigate = useNavigate()
  const seriesId = parseInt(id)

  const { data: series, isLoading, isError, refetch } = useSeriesDetail(seriesId)
  const { data: episodes } = useEpisodes(seriesId)
  const updateMutation = useUpdateSeries()
  const deleteMutation = useDeleteSeries()
  const searchMutation = useSearchSeries()
  const refreshMutation = useRefreshSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()

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

  const handleSearch = async () => {
    try {
      await searchMutation.mutateAsync(seriesId)
      toast.success('Search started')
    } catch {
      toast.error('Failed to start search')
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
      await deleteMutation.mutateAsync(seriesId)
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
          path={undefined} // TODO: Get from metadata
          alt={series.title}
          className="absolute inset-0"
        />
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex gap-6 items-end max-w-4xl">
            {/* Poster */}
            <div className="hidden md:block shrink-0">
              <PosterImage
                path={undefined}
                alt={series.title}
                type="series"
                className="w-40 h-60 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex items-center gap-2">
                <StatusBadge status={series.status} />
                {series.monitored ? (
                  <Badge variant="outline">Monitored</Badge>
                ) : (
                  <Badge variant="secondary">Unmonitored</Badge>
                )}
              </div>
              <h1 className="text-3xl font-bold text-white">{series.title}</h1>
              <div className="flex items-center gap-4 text-sm text-gray-300">
                {series.year && (
                  <span className="flex items-center gap-1">
                    <Calendar className="size-4" />
                    {series.year}
                  </span>
                )}
                {series.runtime && (
                  <span className="flex items-center gap-1">
                    <Clock className="size-4" />
                    {formatRuntime(series.runtime)}
                  </span>
                )}
                {series.sizeOnDisk && (
                  <span className="flex items-center gap-1">
                    <HardDrive className="size-4" />
                    {formatBytes(series.sizeOnDisk)}
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <Tv className="size-4" />
                  {series.episodeFileCount}/{series.episodeCount} episodes
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="px-6 py-4 border-b bg-card flex flex-wrap gap-2">
        <Button onClick={handleSearch} disabled={searchMutation.isPending}>
          <Search className="size-4 mr-2" />
          Search All
        </Button>
        <Button
          variant="outline"
          onClick={handleRefresh}
          disabled={refreshMutation.isPending}
        >
          <RefreshCw className="size-4 mr-2" />
          Refresh
        </Button>
        <Button variant="outline" onClick={handleToggleMonitored}>
          {series.monitored ? (
            <>
              <BookmarkX className="size-4 mr-2" />
              Unmonitor
            </>
          ) : (
            <>
              <Bookmark className="size-4 mr-2" />
              Monitor
            </>
          )}
        </Button>
        <div className="ml-auto flex gap-2">
          <Button variant="outline">
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
        {/* Overview */}
        {series.overview && (
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">{series.overview}</p>
            </CardContent>
          </Card>
        )}

        {/* Details */}
        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Path</span>
              <span className="font-mono text-sm">{series.path || 'Not set'}</span>
            </div>
            <Separator />
            <div className="flex justify-between">
              <span className="text-muted-foreground">Added</span>
              <span>{formatDate(series.addedAt)}</span>
            </div>
            <Separator />
            <div className="flex justify-between">
              <span className="text-muted-foreground">TVDB ID</span>
              <span>{series.tvdbId || '-'}</span>
            </div>
            <Separator />
            <div className="flex justify-between">
              <span className="text-muted-foreground">TMDB ID</span>
              <span>{series.tmdbId || '-'}</span>
            </div>
          </CardContent>
        </Card>

        {/* Seasons */}
        <Card>
          <CardHeader>
            <CardTitle>Seasons & Episodes</CardTitle>
          </CardHeader>
          <CardContent>
            {series.seasons && series.seasons.length > 0 ? (
              <SeasonList
                seasons={series.seasons}
                episodes={episodes}
                onSeasonMonitoredChange={handleSeasonMonitoredChange}
              />
            ) : (
              <p className="text-muted-foreground">No seasons found</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
