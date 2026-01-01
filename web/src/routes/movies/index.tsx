import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Plus, Grid, List, Film, RefreshCw } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { MovieGrid } from '@/components/movies/MovieGrid'
import { MovieTable } from '@/components/movies/MovieTable'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { useMovies, useSearchMovie, useDeleteMovie, useScanLibrary } from '@/hooks'
import { useUIStore } from '@/stores'
import { toast } from 'sonner'
import type { Movie } from '@/types'

type FilterStatus = 'all' | 'monitored' | 'missing' | 'available'

export function MoviesPage() {
  const { moviesView, setMoviesView } = useUIStore()
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')

  const { data: movies, isLoading, isError, refetch } = useMovies()
  const searchMutation = useSearchMovie()
  const deleteMutation = useDeleteMovie()
  const scanMutation = useScanLibrary()

  const handleScanLibrary = async () => {
    try {
      await scanMutation.mutateAsync()
      toast.success('Library scan started')
    } catch {
      toast.error('Failed to start library scan')
    }
  }

  // Filter movies by status
  const filteredMovies = (movies || []).filter((movie: Movie) => {
    if (statusFilter === 'monitored' && !movie.monitored) return false
    if (statusFilter === 'missing' && movie.status !== 'missing') return false
    if (statusFilter === 'available' && movie.status !== 'available') return false
    return true
  })

  const handleSearch = async (id: number) => {
    try {
      await searchMutation.mutateAsync(id)
      toast.success('Search started')
    } catch {
      toast.error('Failed to start search')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Movie deleted')
    } catch {
      toast.error('Failed to delete movie')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Movies" />
        <LoadingState variant={moviesView === 'grid' ? 'card' : 'list'} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Movies" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Movies"
        description={`${movies?.length || 0} movies in library`}
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={handleScanLibrary}
              disabled={scanMutation.isPending}
            >
              <RefreshCw className={`size-4 mr-1 ${scanMutation.isPending ? 'animate-spin' : ''}`} />
              {scanMutation.isPending ? 'Scanning...' : 'Refresh'}
            </Button>
            <Link to="/movies/add">
              <Button>
                <Plus className="size-4 mr-1" />
                Add Movie
              </Button>
            </Link>
          </div>
        }
      />

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <Select
          value={statusFilter}
          onValueChange={(v) => v && setStatusFilter(v as FilterStatus)}
        >
          <SelectTrigger className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="monitored">Monitored</SelectItem>
            <SelectItem value="missing">Missing</SelectItem>
            <SelectItem value="available">Available</SelectItem>
          </SelectContent>
        </Select>

        <div className="ml-auto">
          <ToggleGroup
            value={[moviesView]}
            onValueChange={(v) => v.length > 0 && setMoviesView(v[0] as 'grid' | 'table')}
          >
            <ToggleGroupItem value="grid" aria-label="Grid view">
              <Grid className="size-4" />
            </ToggleGroupItem>
            <ToggleGroupItem value="table" aria-label="Table view">
              <List className="size-4" />
            </ToggleGroupItem>
          </ToggleGroup>
        </div>
      </div>

      {/* Content */}
      {filteredMovies.length === 0 ? (
        <EmptyState
          icon={<Film className="size-8" />}
          title="No movies found"
          description={
            statusFilter !== 'all'
              ? 'Try adjusting your filters'
              : 'Add your first movie to get started'
          }
          action={
            statusFilter === 'all'
              ? { label: 'Add Movie', onClick: () => {} }
              : undefined
          }
        />
      ) : moviesView === 'grid' ? (
        <MovieGrid movies={filteredMovies} />
      ) : (
        <MovieTable
          movies={filteredMovies}
          onSearch={handleSearch}
          onDelete={handleDelete}
        />
      )}
    </div>
  )
}
