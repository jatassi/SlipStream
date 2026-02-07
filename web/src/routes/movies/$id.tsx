import { useState } from 'react'
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
  User,
  Drama,
} from 'lucide-react'
import { BackdropImage } from '@/components/media/BackdropImage'
import { PosterImage } from '@/components/media/PosterImage'
import { TitleTreatment } from '@/components/media/TitleTreatment'
import { StudioLogo } from '@/components/media/StudioLogo'
import { RTFreshIcon, RTRottenIcon, IMDbIcon, MetacriticIcon } from '@/components/media/RatingIcons'
import { MediaStatusBadge } from '@/components/media/MediaStatusBadge'
import { QualityBadge } from '@/components/media/QualityBadge'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { SearchModal } from '@/components/search/SearchModal'
import { AutoSearchButton } from '@/components/search/AutoSearchButton'
import { SlotStatusCard } from '@/components/slots'
import { MovieEditDialog } from '@/components/movies/MovieEditDialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  useMovie,
  useUpdateMovie,
  useDeleteMovie,
  useRefreshMovie,
  useMultiVersionSettings,
  useMovieSlotStatus,
  useSetMovieSlotMonitored,
  useSlots,
  useAssignMovieFile,
  useAutoSearchMovieSlot,
  useQualityProfiles,
  useExtendedMovieMetadata,
} from '@/hooks'
import { formatBytes, formatRuntime, formatDate } from '@/lib/formatters'
import type { Person } from '@/types'
import { toast } from 'sonner'

export function MovieDetailPage() {
  const { id } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const movieId = parseInt(id)

  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [searchQualityProfileId, setSearchQualityProfileId] = useState<number | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)
  const [expandedFileId, setExpandedFileId] = useState<number | null>(null)

  const { data: movie, isLoading, isError, refetch } = useMovie(movieId)
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
  const autoSearchSlotMutation = useAutoSearchMovieSlot()

  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const [searchingSlotId, setSearchingSlotId] = useState<number | null>(null)
  const enabledSlots = slots?.filter(s => s.enabled) ?? []

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) return null
    const slot = slots?.find(s => s.id === slotId)
    return slot?.name ?? null
  }

  const handleToggleMonitored = async () => {
    if (!movie) return
    try {
      await updateMutation.mutateAsync({
        id: movie.id,
        data: { monitored: !movie.monitored },
      })
      toast.success(movie.monitored ? 'Movie unmonitored' : 'Movie monitored')
    } catch {
      toast.error('Failed to update movie')
    }
  }

  const handleManualSearch = () => {
    setSearchModalOpen(true)
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

  const handleSlotManualSearch = (slotId: number) => {
    const slot = slots?.find(s => s.id === slotId)
    if (slot?.qualityProfileId) {
      setSearchQualityProfileId(slot.qualityProfileId)
      setSearchModalOpen(true)
    } else {
      toast.error('Slot has no quality profile configured')
    }
  }

  const handleSlotAutoSearch = async (slotId: number) => {
    const slot = slots?.find(s => s.id === slotId)
    if (!slot?.qualityProfileId) {
      toast.error('Slot has no quality profile configured')
      return
    }

    setSearchingSlotId(slotId)
    try {
      const result = await autoSearchSlotMutation.mutateAsync({ movieId, slotId })
      if (result.downloaded) {
        toast.success(`Release grabbed for ${slot.name}`)
        refetch()
      } else if (result.found) {
        toast.info(`Release found for ${slot.name} but not grabbed`)
      } else {
        toast.info(`No releases found for ${slot.name}`)
      }
    } catch {
      toast.error(`Auto search failed for ${slot.name}`)
    } finally {
      setSearchingSlotId(null)
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
        {movie.studio && (
          <StudioLogo
            tmdbId={movie.tmdbId}
            type="movie"
            alt={movie.studio}
            version={movie.updatedAt}
            className="absolute top-4 right-4 z-10"
            fallback={
              <span className="px-2.5 py-1 rounded bg-black/50 text-xs font-medium text-white/80 backdrop-blur-sm">
                {movie.studio}
              </span>
            }
          />
        )}
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex gap-6 items-end max-w-4xl">
            {/* Poster */}
            <div className="hidden md:block shrink-0">
              <PosterImage
                tmdbId={movie.tmdbId}
                alt={movie.title}
                type="movie"
                version={movie.updatedAt}
                className="w-40 h-60 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex items-center gap-2 flex-wrap">
                <MediaStatusBadge status={movie.status} />
                {qualityProfiles?.find((p) => p.id === movie.qualityProfileId)?.name && (
                  <Badge variant="secondary" className="gap-1">
                    <SlidersVertical className="size-3" />
                    {qualityProfiles.find((p) => p.id === movie.qualityProfileId)?.name}
                  </Badge>
                )}
              </div>
              <TitleTreatment
                tmdbId={movie.tmdbId}
                type="movie"
                alt={movie.title}
                version={movie.updatedAt}
                fallback={<h1 className="text-3xl font-bold text-white">{movie.title}</h1>}
              />
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
                {movie.contentRating && (
                  <span className="shrink-0 px-1.5 py-0.5 border border-gray-400 rounded text-xs font-medium text-gray-300">
                    {movie.contentRating}
                  </span>
                )}
                {movie.year && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Calendar className="size-4 shrink-0" />
                    {movie.year}
                  </span>
                )}
                {movie.runtime && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Clock className="size-4 shrink-0" />
                    {formatRuntime(movie.runtime)}
                  </span>
                )}
                {extendedData?.credits?.directors?.[0] && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserStar className="size-4 shrink-0" />
                    {extendedData.credits.directors[0].name}
                  </span>
                )}
                {extendedData?.genres && extendedData.genres.length > 0 && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <Drama className="size-4 shrink-0" />
                    {extendedData.genres.join(', ')}
                  </span>
                )}
                {movie.addedByUsername && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <UserRoundPlus className="size-4 shrink-0" />
                    {movie.addedByUsername}
                  </span>
                )}
                {movie.addedAt && (
                  <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
                    <CalendarPlus className="size-4 shrink-0" />
                    {formatDate(movie.addedAt)}
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
              {movie.overview && (
                <p
                  className={`text-sm text-gray-300 max-w-2xl cursor-pointer ${overviewExpanded ? '' : 'line-clamp-2'}`}
                  onClick={() => setOverviewExpanded(!overviewExpanded)}
                >
                  {movie.overview}
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
          mediaType="movie"
          movieId={movie.id}
          title={movie.title}
        />
        <Button variant="outline" onClick={handleToggleMonitored} className={movie.monitored ? 'glow-movie-sm' : ''}>
          {movie.monitored ? (
            <>
              <Eye className="size-4 mr-2 text-movie-400" />
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
            title="Delete movie"
            description={`Are you sure you want to delete "${movie.title}"? This action cannot be undone.`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={handleDelete}
          />
        </div>
      </div>

      {/* Content */}
      <div className="p-6 space-y-6">
        {/* Multi-Version Slot Status */}
        {isMultiVersionEnabled && (
          <SlotStatusCard
            status={slotStatus}
            isLoading={isLoadingSlotStatus}
            onToggleMonitored={handleToggleSlotMonitored}
            onManualSearch={handleSlotManualSearch}
            onAutoSearch={handleSlotAutoSearch}
            isUpdating={setSlotMonitoredMutation.isPending}
            isSearching={searchingSlotId}
          />
        )}

        {/* Files */}
        <Card>
          <CardHeader>
            <CardTitle>Files</CardTitle>
          </CardHeader>
          <CardContent>
            {!movie.movieFiles?.length ? (
              <p className="text-muted-foreground">No files found</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Filename</TableHead>
                    {expandedFileId == null && (
                      <>
                        <TableHead>Quality</TableHead>
                        <TableHead>Video</TableHead>
                        <TableHead>Audio</TableHead>
                        {isMultiVersionEnabled && <TableHead>Slot</TableHead>}
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
                                {file.videoCodec && (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.videoCodec}
                                  </Badge>
                                )}
                                {file.dynamicRange && file.dynamicRange.split(' ').map((dr) => (
                                  <Badge key={dr} variant="outline" className="font-mono text-xs">
                                    {dr}
                                  </Badge>
                                ))}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1">
                                {file.audioCodec && (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.audioCodec}
                                  </Badge>
                                )}
                                {file.audioChannels && (
                                  <Badge variant="outline" className="font-mono text-xs">
                                    {file.audioChannels}
                                  </Badge>
                                )}
                              </div>
                            </TableCell>
                            {isMultiVersionEnabled && (
                              <TableCell>
                                <Select
                                  value={file.slotId?.toString() ?? 'unassigned'}
                                  onValueChange={(value) => {
                                    if (value && value !== 'unassigned') {
                                      handleAssignFileToSlot(file.id, parseInt(value, 10))
                                    }
                                  }}
                                  disabled={assignFileMutation.isPending}
                                >
                                  <SelectTrigger className="w-32 h-8">
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
                            )}
                            <TableCell className="text-right">{formatBytes(file.size)}</TableCell>
                          </>
                        )}
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        {/* Cast */}
        {extendedData?.credits?.cast && extendedData.credits.cast.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Cast</CardTitle>
            </CardHeader>
            <CardContent>
              <PersonList people={extendedData.credits.cast} max={18} />
            </CardContent>
          </Card>
        )}

        {/* Crew */}
        {extendedData?.credits && (extendedData.credits.directors?.length || extendedData.credits.writers?.length) && (
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
        )}
      </div>

      {/* Search Modal */}
      <SearchModal
        open={searchModalOpen}
        onOpenChange={(open) => {
          setSearchModalOpen(open)
          if (!open) setSearchQualityProfileId(null)
        }}
        qualityProfileId={searchQualityProfileId ?? movie.qualityProfileId}
        movieId={movie.id}
        movieTitle={movie.title}
        tmdbId={movie.tmdbId}
        imdbId={movie.imdbId}
        year={movie.year}
      />

      {/* Edit Dialog */}
      <MovieEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        movie={movie}
      />
    </div>
  )
}

function PersonList({ people, max = 12 }: { people: Person[]; max?: number }) {
  return (
    <div className="flex gap-4 overflow-x-auto pb-2">
      {people.slice(0, max).map((person) => (
        <div
          key={`${person.id}-${person.role}`}
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
  )
}
