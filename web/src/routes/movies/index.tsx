import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Plus, Grid, List, Film, RefreshCw, Pencil, Trash2, X } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { MovieGrid } from '@/components/movies/MovieGrid'
import { MovieTable } from '@/components/movies/MovieTable'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { useMovies, useSearchMovie, useDeleteMovie, useBulkDeleteMovies, useScanLibrary } from '@/hooks'
import { useUIStore } from '@/stores'
import { toast } from 'sonner'
import type { Movie } from '@/types'

type FilterStatus = 'all' | 'monitored' | 'missing' | 'available'

export function MoviesPage() {
  const { moviesView, setMoviesView } = useUIStore()
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')
  const [editMode, setEditMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)

  const { data: movies, isLoading, isError, refetch } = useMovies()
  const searchMutation = useSearchMovie()
  const deleteMutation = useDeleteMovie()
  const bulkDeleteMutation = useBulkDeleteMovies()
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
      await deleteMutation.mutateAsync({ id })
      toast.success('Movie deleted')
    } catch {
      toast.error('Failed to delete movie')
    }
  }

  const handleToggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const handleSelectAll = () => {
    if (selectedIds.size === filteredMovies.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredMovies.map((m) => m.id)))
    }
  }

  const handleExitEditMode = () => {
    setEditMode(false)
    setSelectedIds(new Set())
  }

  const handleBulkDelete = async () => {
    try {
      await bulkDeleteMutation.mutateAsync({
        ids: Array.from(selectedIds),
        deleteFiles,
      })
      toast.success(`${selectedIds.size} movie${selectedIds.size > 1 ? 's' : ''} deleted`)
      setShowDeleteDialog(false)
      setDeleteFiles(false)
      handleExitEditMode()
    } catch {
      toast.error('Failed to delete movies')
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
              disabled={scanMutation.isPending || editMode}
            >
              <RefreshCw className={`size-4 mr-1 ${scanMutation.isPending ? 'animate-spin' : ''}`} />
              {scanMutation.isPending ? 'Scanning...' : 'Refresh'}
            </Button>
            {editMode ? (
              <Button variant="outline" onClick={handleExitEditMode}>
                <X className="size-4 mr-1" />
                Cancel
              </Button>
            ) : (
              <Button variant="outline" onClick={() => setEditMode(true)}>
                <Pencil className="size-4 mr-1" />
                Edit
              </Button>
            )}
            <Link to="/movies/add">
              <Button disabled={editMode}>
                <Plus className="size-4 mr-1" />
                Add Movie
              </Button>
            </Link>
          </div>
        }
      />

      {/* Edit Mode Toolbar */}
      {editMode && (
        <div className="flex items-center gap-4 mb-4 p-3 bg-muted rounded-lg">
          <div className="flex items-center gap-2">
            <Checkbox
              checked={selectedIds.size === filteredMovies.length && filteredMovies.length > 0}
              onCheckedChange={handleSelectAll}
            />
            <span className="text-sm text-muted-foreground">
              {selectedIds.size} of {filteredMovies.length} selected
            </span>
          </div>
          <div className="ml-auto">
            <Button
              variant="destructive"
              size="sm"
              disabled={selectedIds.size === 0}
              onClick={() => setShowDeleteDialog(true)}
            >
              <Trash2 className="size-4 mr-1" />
              Delete Selected
            </Button>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <Select
          value={statusFilter}
          onValueChange={(v) => v && setStatusFilter(v as FilterStatus)}
        >
          <SelectTrigger className="w-36">
            {{ all: 'All', monitored: 'Monitored', missing: 'Missing', available: 'Available' }[statusFilter]}
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
        <MovieGrid
          movies={filteredMovies}
          editMode={editMode}
          selectedIds={selectedIds}
          onToggleSelect={handleToggleSelect}
        />
      ) : (
        <MovieTable
          movies={filteredMovies}
          onSearch={handleSearch}
          onDelete={handleDelete}
        />
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {selectedIds.size} Movie{selectedIds.size > 1 ? 's' : ''}?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. The selected movies will be removed from your library.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 py-2">
            <Checkbox
              id="deleteFiles"
              checked={deleteFiles}
              onCheckedChange={(checked) => setDeleteFiles(checked === true)}
            />
            <label htmlFor="deleteFiles" className="text-sm cursor-pointer">
              Also delete files from disk
            </label>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleBulkDelete}
              disabled={bulkDeleteMutation.isPending}
            >
              {bulkDeleteMutation.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
