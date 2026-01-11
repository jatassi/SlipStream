import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Search, Zap, Film, Loader2, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SearchModal } from '@/components/search/SearchModal'
import { EmptyState } from '@/components/data/EmptyState'
import { formatDate } from '@/lib/formatters'
import { useAutoSearchMovie } from '@/hooks'
import { useDownloadingStore } from '@/stores'
import { toast } from 'sonner'
import type { MissingMovie } from '@/types/missing'
import type { AutoSearchResult } from '@/types'

interface MissingMoviesListProps {
  movies: MissingMovie[]
  isSearchingAll?: boolean
}

export function MissingMoviesList({ movies, isSearchingAll = false }: MissingMoviesListProps) {
  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [selectedMovie, setSelectedMovie] = useState<MissingMovie | null>(null)
  const [searchingMovieId, setSearchingMovieId] = useState<number | null>(null)

  const autoSearchMutation = useAutoSearchMovie()

  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isMovieDownloading = (movieId: number) => {
    return queueItems.some(
      (item) =>
        item.movieId === movieId &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }

  const handleManualSearch = (movie: MissingMovie) => {
    setSelectedMovie(movie)
    setSearchModalOpen(true)
  }

  const formatResult = (result: AutoSearchResult, title: string) => {
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

  const handleAutoSearch = async (movie: MissingMovie) => {
    setSearchingMovieId(movie.id)
    try {
      const result = await autoSearchMutation.mutateAsync(movie.id)
      formatResult(result, movie.title)
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning(`"${movie.title}" is already in the download queue`)
      } else {
        toast.error(`Search failed for "${movie.title}"`)
      }
    } finally {
      setSearchingMovieId(null)
    }
  }

  if (movies.length === 0) {
    return (
      <EmptyState
        icon={<Film className="size-8" />}
        title="No missing movies"
        description="All monitored movies with available release dates have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead className="w-[120px]">Year</TableHead>
            <TableHead className="w-[150px]">Release Date</TableHead>
            <TableHead className="w-[120px] text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {movies.map((movie) => (
            <TableRow key={movie.id}>
              <TableCell>
                <Link
                  to="/movies/$id"
                  params={{ id: movie.id.toString() }}
                  className="font-medium hover:underline"
                >
                  {movie.title}
                </Link>
              </TableCell>
              <TableCell className="text-muted-foreground">
                {movie.year || '-'}
              </TableCell>
              <TableCell className="text-muted-foreground">
                {movie.releaseDate ? formatDate(movie.releaseDate) : '-'}
              </TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-1">
                  {isMovieDownloading(movie.id) ? (
                    <Button
                      variant="ghost"
                      size="icon"
                      disabled
                      title="Downloading"
                    >
                      <Download className="size-4 text-green-500" />
                    </Button>
                  ) : (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleAutoSearch(movie)}
                      disabled={isSearchingAll || searchingMovieId === movie.id}
                      title="Automatic Search"
                    >
                      {isSearchingAll || searchingMovieId === movie.id ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : (
                        <Zap className="size-4" />
                      )}
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleManualSearch(movie)}
                    title="Manual Search"
                  >
                    <Search className="size-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {selectedMovie && (
        <SearchModal
          open={searchModalOpen}
          onOpenChange={setSearchModalOpen}
          qualityProfileId={selectedMovie.qualityProfileId}
          movieId={selectedMovie.id}
          movieTitle={selectedMovie.title}
          tmdbId={selectedMovie.tmdbId}
          imdbId={selectedMovie.imdbId}
          year={selectedMovie.year}
        />
      )}
    </>
  )
}
