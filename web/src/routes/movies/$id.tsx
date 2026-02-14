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
  User,
  UserRoundPlus,
  UserStar,
} from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { BackdropImage } from '@/components/media/BackdropImage'
import { MediaStatusBadge } from '@/components/media/MediaStatusBadge'
import { PosterImage } from '@/components/media/PosterImage'
import { QualityBadge } from '@/components/media/QualityBadge'
import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/RatingIcons'
import { StudioLogo } from '@/components/media/StudioLogo'
import { TitleTreatment } from '@/components/media/TitleTreatment'
import { MovieEditDialog } from '@/components/movies/MovieEditDialog'
import { MediaSearchMonitorControls } from '@/components/search'
import { SlotStatusCard } from '@/components/slots'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  useAssignMovieFile,
  useDeleteMovie,
  useExtendedMovieMetadata,
  useGlobalLoading,
  useMovie,
  useMovieSlotStatus,
  useMultiVersionSettings,
  useQualityProfiles,
  useRefreshMovie,
  useSetMovieSlotMonitored,
  useSlots,
  useUpdateMovie,
} from '@/hooks'
import { formatBytes, formatDate, formatRuntime } from '@/lib/formatters'
import type { Person } from '@/types'

export function MovieDetailPage() {
  const { id } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const movieId = Number.parseInt(id)

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)
  const [expandedFileId, setExpandedFileId] = useState<number | null>(null)

  const globalLoading = useGlobalLoading()
  const { data: movie, isLoading: queryLoading, isError, refetch } = useMovie(movieId)
  const isLoading = queryLoading || globalLoading
  const { data: extendedData } = useExtendedMovieMetadata(movie?.tmdbId ?? 0)
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slotStatus, isLoading: isLoadingSlotStatus } = useMovieSlotStatus(movieId)
  const { data: slots } = useSlots()
  const updateMutation = useUpdateMovie()
  const deleteMutation = useDeleteMovie()
  const refreshMutation = useRefreshMovie()
  const setSlotMonitoredMutation = useSetMovieSlotMonitored()
  const assignFileMutation = useAssignMovieFile()

  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = useMemo(() => slots?.filter((s) => s.enabled) ?? [], [slots])

  const slotQualityProfiles = useMemo(() => {
    const map: Record<number, number> = {}
    for (const slot of enabledSlots) {
      if (slot.qualityProfileId != null) {
        map[slot.id] = slot.qualityProfileId
      }
    }
    return map
  }, [enabledSlots])

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) {
      return null
    }
    const slot = slots?.find((s) => s.id === slotId)
    return slot?.name ?? null
  }

  const handleToggleMonitored = async (newMonitored?: boolean) => {
    if (!movie) {
      return
    }
    const target = newMonitored ?? !movie.monitored
    try {
      await updateMutation.mutateAsync({
        id: movie.id,
        data: { monitored: target },
      })
      toast.success(target ? 'Movie monitored' : 'Movie unmonitored')
    } catch {
      toast.error('Failed to update movie')
    }
  }

  const handleRefresh = async () => {
    try {
      await refreshMutation.mutateAsync(movieId)
      toast.success('Metadata refreshed')
    } catch {
      toast.error('Failed to refresh metadata')
    }
  }

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync({ id: movieId })
      toast.success('Movie deleted')
      navigate({ to: '/movies' })
    } catch {
      toast.error('Failed to delete movie')
    }
  }

  const handleToggleSlotMonitored = async (slotId: number, monitored: boolean) => {
    try {
      await setSlotMonitoredMutation.mutateAsync({
        movieId,
        slotId,
        data: { monitored },
      })
      toast.success(monitored ? 'Slot monitored' : 'Slot unmonitored')
    } catch {
      toast.error('Failed to update slot monitoring')
    }
  }

  const handleAssignFileToSlot = async (fileId: number, slotId: number) => {
    try {
      await assignFileMutation.mutateAsync({
        movieId,
        slotId,
        data: { fileId },
      })
      refetch()
      toast.success('File assigned to slot')
    } catch {
      toast.error('Failed to assign file to slot')
    }
  }

  if (isLoading) {
    return <LoadingState variant="detail" />
  }

  if (isError || !movie) {
    return <ErrorState message="Movie not found" onRetry={refetch} />
  }

  return (
    <div className="-m-6">
      {/* Hero with backdrop */}
      <div className="relative h-64 md:h-80">
        <BackdropImage
          tmdbId={movie.tmdbId}
          type="movie"
          alt={movie.title}
          version={movie.updatedAt}
          className="absolute inset-0"
        />
        {movie.studio ? (
          <StudioLogo
            tmdbId={movie.tmdbId}
            type="movie"
            alt={movie.studio}
            version={movie.updatedAt}
            className="absolute top-4 right-4 z-10"
            fallback={
              <span className="rounded bg-black/50 px-2.5 py-1 text-xs font-medium text-white/80 backdrop-blur-sm">
                {movie.studio}
              </span>
            }
          />
        ) : null}
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex max-w-4xl items-end gap-6">
            {/* Poster */}
            <div className="hidden shrink-0 md:block">
              <PosterImage
                tmdbId={movie.tmdbId}
                alt={movie.title}
                type="movie"
                version={movie.updatedAt}
                className="h-60 w-40 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                <MediaStatusBadge status={movie.status} />
                {qualityProfiles?.find((p) => p.id === movie.qualityProfileId)?.name ? (
                  <Badge variant="secondary" className="gap-1">
                    <SlidersVertical className="size-3" />
                    {qualityProfiles.find((p) => p.id === movie.qualityProfileId)?.name}
                  </Badge>
                ) : null}
              </div>
              <TitleTreatment
                tmdbId={movie.tmdbId}
                type="movie"
                alt={movie.title}
                version={movie.updatedAt}
                fallback={<h1 className="text-3xl font-bold text-white">{movie.title}</h1>}
              />
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
                {movie.contentRating ? (
                  <span className="shrink-0 rounded border border-gray-400 px-1.5 py-0.5 text-xs font-medium text-gray-300">
                    {movie.contentRating}
                  </span>
                ) : null}
                {movie.year ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Calendar className="size-4 shrink-0" />
                    {movie.year}
                  </span>
                ) : null}
                {movie.runtime ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Clock className="size-4 shrink-0" />
                    {formatRuntime(movie.runtime)}
                  </span>
                ) : null}
                {extendedData?.credits?.directors?.[0] ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserStar className="size-4 shrink-0" />
                    {extendedData.credits.directors[0].name}
                  </span>
                ) : null}
                {extendedData?.genres && extendedData.genres.length > 0 ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Drama className="size-4 shrink-0" />
                    {extendedData.genres.join(', ')}
                  </span>
                ) : null}
                {movie.addedByUsername ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserRoundPlus className="size-4 shrink-0" />
                    {movie.addedByUsername}
                  </span>
                ) : null}
                {movie.addedAt ? (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <CalendarPlus className="size-4 shrink-0" />
                    {formatDate(movie.addedAt)}
                  </span>
                ) : null}
              </div>
              {(extendedData?.ratings?.rottenTomatoes != null ||
                extendedData?.ratings?.imdbRating != null ||
                extendedData?.ratings?.metacritic != null) && (
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
                      <span className="font-medium">
                        {extendedData.ratings.imdbRating.toFixed(1)}
                      </span>
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
              {movie.overview ? (
                <p
                  className={`max-w-2xl cursor-pointer text-sm text-gray-300 ${overviewExpanded ? '' : 'line-clamp-2'}`}
                  onClick={() => setOverviewExpanded(!overviewExpanded)}
                >
                  {movie.overview}
                </p>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="bg-card flex flex-wrap gap-2 border-b px-6 py-4">
        <MediaSearchMonitorControls
          mediaType="movie"
          movieId={movie.id}
          title={movie.title}
          theme="movie"
          size="responsive"
          monitored={movie.monitored}
          onMonitoredChange={handleToggleMonitored}
          qualityProfileId={movie.qualityProfileId}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          year={movie.year}
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
            title="Delete movie"
            description={`Are you sure you want to delete "${movie.title}"? This action cannot be undone.`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={handleDelete}
          />
        </div>
      </div>

      {/* Content */}
      <div className="space-y-6 p-6">
        {/* Multi-Version Slot Status */}
        {isMultiVersionEnabled ? (
          <SlotStatusCard
            status={slotStatus}
            isLoading={isLoadingSlotStatus}
            movieId={movieId}
            movieTitle={movie.title}
            qualityProfileId={movie.qualityProfileId}
            tmdbId={movie.tmdbId}
            imdbId={movie.imdbId}
            year={movie.year}
            slotQualityProfiles={slotQualityProfiles}
            onToggleMonitored={handleToggleSlotMonitored}
            isUpdating={setSlotMonitoredMutation.isPending}
          />
        ) : null}

        {/* Files */}
        <Card>
          <CardHeader>
            <CardTitle>Files</CardTitle>
          </CardHeader>
          <CardContent>
            {movie.movieFiles?.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Filename</TableHead>
                    {expandedFileId == null && (
                      <>
                        <TableHead>Quality</TableHead>
                        <TableHead>Video</TableHead>
                        <TableHead>Audio</TableHead>
                        {isMultiVersionEnabled ? <TableHead>Slot</TableHead> : null}
                        <TableHead className="text-right">Size</TableHead>
                      </>
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {movie.movieFiles.map((file) => {
                    const filename = file.path.split('/').pop() || file.path
                    const isExpanded = expandedFileId === file.id
                    return (
                      <TableRow key={file.id}>
                        <TableCell
                          className="cursor-pointer"
                          onClick={() => setExpandedFileId(isExpanded ? null : file.id)}
                        >
                          {isExpanded ? (
                            <span className="font-mono text-xs break-all">{file.path}</span>
                          ) : (
                            <span className="font-mono text-sm">{filename}</span>
                          )}
                        </TableCell>
                        {!isExpanded && expandedFileId == null && (
                          <>
                            <TableCell>
                              <QualityBadge quality={file.quality} />
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1">
                                {file.videoCodec ? (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.videoCodec}
                                  </Badge>
                                ) : null}
                                {file.dynamicRange?.split(' ').map((dr) => (
                                  <Badge key={dr} variant="outline" className="font-mono text-xs">
                                    {dr}
                                  </Badge>
                                ))}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1">
                                {file.audioCodec ? (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.audioCodec}
                                  </Badge>
                                ) : null}
                                {file.audioChannels ? (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.audioChannels}
                                  </Badge>
                                ) : null}
                              </div>
                            </TableCell>
                            {isMultiVersionEnabled ? (
                              <TableCell>
                                <Select
                                  value={file.slotId?.toString() ?? 'unassigned'}
                                  onValueChange={(value) => {
                                    if (value && value !== 'unassigned') {
                                      handleAssignFileToSlot(file.id, Number.parseInt(value, 10))
                                    }
                                  }}
                                  disabled={assignFileMutation.isPending}
                                >
                                  <SelectTrigger className="h-8 w-32">
                                    {getSlotName(file.slotId) ?? (
                                      <span className="text-muted-foreground">Unassigned</span>
                                    )}
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="unassigned" disabled>
                                      Unassigned
                                    </SelectItem>
                                    {enabledSlots.map((slot) => (
                                      <SelectItem key={slot.id} value={slot.id.toString()}>
                                        {slot.name}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              </TableCell>
                            ) : null}
                            <TableCell className="text-right">{formatBytes(file.size)}</TableCell>
                          </>
                        )}
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            ) : (
              <p className="text-muted-foreground">No files found</p>
            )}
          </CardContent>
        </Card>

        {/* Cast */}
        {extendedData?.credits?.cast && extendedData.credits.cast.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Cast</CardTitle>
            </CardHeader>
            <CardContent>
              <PersonList people={extendedData.credits.cast} max={18} />
            </CardContent>
          </Card>
        ) : null}

        {/* Crew */}
        {extendedData?.credits &&
        (extendedData.credits.directors?.length || extendedData.credits.writers?.length) ? (
          <Card>
            <CardHeader>
              <CardTitle>Crew</CardTitle>
            </CardHeader>
            <CardContent>
              <PersonList
                people={[
                  ...(extendedData.credits.directors ?? []),
                  ...(extendedData.credits.writers ?? []),
                ]}
                max={12}
              />
            </CardContent>
          </Card>
        ) : null}
      </div>

      {/* Edit Dialog */}
      <MovieEditDialog open={editDialogOpen} onOpenChange={setEditDialogOpen} movie={movie} />
    </div>
  )
}

function PersonList({ people, max = 12 }: { people: Person[]; max?: number }) {
  return (
    <div className="flex gap-4 overflow-x-auto pb-2">
      {people.slice(0, max).map((person) => (
        <div
          key={`${person.id}-${person.role}`}
          className="flex w-20 shrink-0 flex-col items-center gap-1"
        >
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
  )
}
