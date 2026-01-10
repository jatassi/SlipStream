import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Search, Zap, Film } from 'lucide-react'
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
import type { MissingMovie } from '@/types/missing'

interface MissingMoviesListProps {
  movies: MissingMovie[]
}

export function MissingMoviesList({ movies }: MissingMoviesListProps) {
  const [searchModalOpen, setSearchModalOpen] = useState(false)
  const [selectedMovie, setSelectedMovie] = useState<MissingMovie | null>(null)

  const handleManualSearch = (movie: MissingMovie) => {
    setSelectedMovie(movie)
    setSearchModalOpen(true)
  }

  const handleAutoSearch = (movie: MissingMovie) => {
    // Placeholder for automatic search - will be wired up later
    console.log('Auto search for movie:', movie.title)
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
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleAutoSearch(movie)}
                    title="Automatic Search"
                  >
                    <Zap className="size-4" />
                  </Button>
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
