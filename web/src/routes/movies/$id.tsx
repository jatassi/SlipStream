import { useState } from 'react'
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
  Zap,
} from 'lucide-react'
import { BackdropImage } from '@/components/media/BackdropImage'
import { PosterImage } from '@/components/media/PosterImage'
import { StatusBadge } from '@/components/media/StatusBadge'
import { MovieAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { QualityBadge } from '@/components/media/QualityBadge'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { SearchModal } from '@/components/search/SearchModal'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  useMovie,
  useUpdateMovie,
  useDeleteMovie,
  useSearchMovie,
  useRefreshMovie,
} from '@/hooks'
import { formatBytes, formatRuntime, formatDate } from '@/lib/formatters'
import { toast } from 'sonner'

export function MovieDetailPage() {
  const { id } = useParams({ from: '/movies/$id' })
  const navigate = useNavigate()
  const movieId = parseInt(id)

  const [searchModalOpen, setSearchModalOpen] = useState(false)

  const { data: movie, isLoading, isError, refetch } = useMovie(movieId)
  const updateMutation = useUpdateMovie()
  const deleteMutation = useDeleteMovie()
  const searchMutation = useSearchMovie()
  const refreshMutation = useRefreshMovie()

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

  const handleAutoSearch = async () => {
    try {
      await searchMutation.mutateAsync(movieId)
      toast.success('Automatic search started')
    } catch {
      toast.error('Failed to start search')
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
      await deleteMutation.mutateAsync(movieId)
      toast.success('Movie deleted')
      navigate({ to: '/movies' })
    } catch {
      toast.error('Failed to delete movie')
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
          className="absolute inset-0"
        />
        <div className="absolute inset-0 flex items-end p-6">
          <div className="flex gap-6 items-end max-w-4xl">
            {/* Poster */}
            <div className="hidden md:block shrink-0">
              <PosterImage
                tmdbId={movie.tmdbId}
                alt={movie.title}
                type="movie"
                className="w-40 h-60 rounded-lg shadow-lg"
              />
            </div>

            {/* Info */}
            <div className="flex-1 space-y-2">
              <div className="flex items-center gap-2">
                <StatusBadge status={movie.status} />
                <MovieAvailabilityBadge movie={movie} />
                {movie.monitored ? (
                  <Badge variant="outline">Monitored</Badge>
                ) : (
                  <Badge variant="secondary">Unmonitored</Badge>
                )}
              </div>
              <h1 className="text-3xl font-bold text-white">{movie.title}</h1>
              <div className="flex items-center gap-4 text-sm text-gray-300">
                {(movie.releaseDate || movie.year) && (
                  <span className="flex items-center gap-1">
                    <Calendar className="size-4" />
                    {movie.releaseDate ? formatDate(movie.releaseDate) : movie.year}
                  </span>
                )}
                {movie.runtime && (
                  <span className="flex items-center gap-1">
                    <Clock className="size-4" />
                    {formatRuntime(movie.runtime)}
                  </span>
                )}
                {movie.sizeOnDisk && (
                  <span className="flex items-center gap-1">
                    <HardDrive className="size-4" />
                    {formatBytes(movie.sizeOnDisk)}
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="px-6 py-4 border-b bg-card flex flex-wrap gap-2">
        <Button onClick={handleManualSearch}>
          <Search className="size-4 mr-2" />
          Search
        </Button>
        <Button
          variant="outline"
          onClick={handleAutoSearch}
          disabled={searchMutation.isPending}
        >
          <Zap className="size-4 mr-2" />
          Auto Search
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
          {movie.monitored ? (
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
        {/* Overview */}
        {movie.overview && (
          <Card>
            <CardHeader>
              <CardTitle>Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">{movie.overview}</p>
            </CardContent>
          </Card>
        )}

        {/* Details */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Path</span>
                <span className="font-mono text-sm">{movie.path || 'Not set'}</span>
              </div>
              <Separator />
              <div className="flex justify-between">
                <span className="text-muted-foreground">Added</span>
                <span>{formatDate(movie.addedAt)}</span>
              </div>
              <Separator />
              <div className="flex justify-between">
                <span className="text-muted-foreground">TMDB ID</span>
                <span>{movie.tmdbId || '-'}</span>
              </div>
              <Separator />
              <div className="flex justify-between">
                <span className="text-muted-foreground">IMDB ID</span>
                <span>{movie.imdbId || '-'}</span>
              </div>
            </CardContent>
          </Card>

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
                      <TableHead>Quality</TableHead>
                      <TableHead>Size</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {movie.movieFiles.map((file) => (
                      <TableRow key={file.id}>
                        <TableCell>
                          <QualityBadge quality={file.quality} />
                        </TableCell>
                        <TableCell>{formatBytes(file.size)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Search Modal */}
      <SearchModal
        open={searchModalOpen}
        onOpenChange={setSearchModalOpen}
        movieId={movie.id}
        movieTitle={movie.title}
        tmdbId={movie.tmdbId}
        imdbId={movie.imdbId}
        year={movie.year}
      />
    </div>
  )
}
