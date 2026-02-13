import { useState, useMemo } from 'react'
import { Plus, Grid, List, Film, RefreshCw, Pencil, Trash2, X, Eye, EyeOff, ArrowUpDown, Clock, Binoculars, ArrowDownCircle, XCircle, ArrowUpCircle, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Checkbox } from '@/components/ui/checkbox'
import { Slider } from '@/components/ui/slider'
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
import { GroupedMovieGrid } from '@/components/movies/GroupedMovieGrid'
import { MovieTable } from '@/components/movies/MovieTable'
import { ColumnConfigPopover } from '@/components/tables/ColumnConfigPopover'
import { LoadingState } from '@/components/data/LoadingState'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { groupMedia } from '@/lib/grouping'
import { MOVIE_COLUMNS, createMovieActionsColumn, DEFAULT_SORT_DIRECTIONS } from '@/lib/table-columns'
import { useMovies, useSearchMovie, useDeleteMovie, useBulkDeleteMovies, useBulkUpdateMovies, useRefreshAllMovies, useQualityProfiles, useRootFolders, useGlobalLoading } from '@/hooks'
import { useUIStore } from '@/stores'
import { toast } from 'sonner'
import type { Movie } from '@/types'

type FilterStatus = 'monitored' | 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
type SortField = 'title' | 'monitored' | 'qualityProfile' | 'releaseDate' | 'dateAdded' | 'rootFolder' | 'sizeOnDisk'

const ALL_FILTERS: FilterStatus[] = ['monitored', 'unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available']

const FILTER_OPTIONS: { value: FilterStatus; label: string; icon: typeof Eye }[] = [
  { value: 'monitored', label: 'Monitored', icon: Eye },
  { value: 'unreleased', label: 'Unreleased', icon: Clock },
  { value: 'missing', label: 'Missing', icon: Binoculars },
  { value: 'downloading', label: 'Downloading', icon: ArrowDownCircle },
  { value: 'failed', label: 'Failed', icon: XCircle },
  { value: 'upgradable', label: 'Upgradable', icon: ArrowUpCircle },
  { value: 'available', label: 'Available', icon: CheckCircle },
]

const SORT_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'title', label: 'Title' },
  { value: 'monitored', label: 'Monitored' },
  { value: 'qualityProfile', label: 'Quality Profile' },
  { value: 'releaseDate', label: 'Release Date' },
  { value: 'dateAdded', label: 'Date Added' },
  { value: 'rootFolder', label: 'Root Folder' },
  { value: 'sizeOnDisk', label: 'Size on Disk' },
]

export function MoviesPage() {
  const { moviesView, setMoviesView, posterSize, setPosterSize, movieTableColumns, setMovieTableColumns } = useUIStore()
  const [statusFilters, setStatusFilters] = useState<FilterStatus[]>([...ALL_FILTERS])
  const [sortField, setSortField] = useState<SortField>('title')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [editMode, setEditMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)

  const globalLoading = useGlobalLoading()
  const { data: movies, isLoading: queryLoading, isError, refetch } = useMovies()
  const isLoading = queryLoading || globalLoading
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: rootFolders } = useRootFolders()
  const searchMutation = useSearchMovie()
  const deleteMutation = useDeleteMovie()
  const bulkDeleteMutation = useBulkDeleteMovies()
  const bulkUpdateMutation = useBulkUpdateMovies()
  const refreshAllMutation = useRefreshAllMovies()

  const handleRefreshAll = async () => {
    try {
      await refreshAllMutation.mutateAsync()
      toast.success('Refresh started for all movies')
    } catch {
      toast.error('Failed to start refresh')
    }
  }

  const handleColumnSort = (field: string) => {
    if (field === sortField) {
      setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field as SortField)
      setSortDirection(DEFAULT_SORT_DIRECTIONS[field] || 'asc')
    }
  }

  const profileNameMap = new Map(qualityProfiles?.map((p) => [p.id, p.name]) || [])
  const rootFolderNameMap = new Map(rootFolders?.map((f) => [f.id, f.name]) || [])

  const renderContext = { qualityProfileNames: profileNameMap, rootFolderNames: rootFolderNameMap }

  const allFiltersSelected = statusFilters.length >= ALL_FILTERS.length

  // Filter movies by status
  const filteredMovies = (movies || []).filter((movie: Movie) => {
    if (allFiltersSelected) return true
    if (statusFilters.includes('monitored') && movie.monitored) return true
    if (statusFilters.includes(movie.status as FilterStatus)) return true
    return false
  })

  // Sort movies
  const defaultDir = DEFAULT_SORT_DIRECTIONS[sortField] || 'asc'
  const dirMultiplier = sortDirection === defaultDir ? 1 : -1
  const sortedMovies = [...filteredMovies].sort((a, b) => {
    let result: number
    switch (sortField) {
      case 'monitored':
        result = (b.monitored ? 1 : 0) - (a.monitored ? 1 : 0) || a.sortTitle.localeCompare(b.sortTitle)
        break
      case 'qualityProfile': {
        const nameA = profileNameMap.get(a.qualityProfileId) || ''
        const nameB = profileNameMap.get(b.qualityProfileId) || ''
        result = nameA.localeCompare(nameB) || a.sortTitle.localeCompare(b.sortTitle)
        break
      }
      case 'releaseDate': {
        const dateA = a.releaseDate ?? a.physicalReleaseDate ?? a.theatricalReleaseDate
        const dateB = b.releaseDate ?? b.physicalReleaseDate ?? b.theatricalReleaseDate
        if (!dateA && !dateB) { result = a.sortTitle.localeCompare(b.sortTitle); break }
        if (!dateA) { result = 1; break }
        if (!dateB) { result = -1; break }
        result = new Date(dateB).getTime() - new Date(dateA).getTime()
        break
      }
      case 'dateAdded':
        result = new Date(b.addedAt).getTime() - new Date(a.addedAt).getTime()
        break
      case 'rootFolder':
        result = (a.rootFolderId || 0) - (b.rootFolderId || 0) || a.sortTitle.localeCompare(b.sortTitle)
        break
      case 'sizeOnDisk':
        result = (b.sizeOnDisk || 0) - (a.sizeOnDisk || 0)
        break
      default:
        result = a.sortTitle.localeCompare(b.sortTitle)
    }
    return result * dirMultiplier
  })

  const groups = groupMedia(
    sortedMovies.map((m) => ({
      ...m,
      releaseDate: m.releaseDate ?? m.physicalReleaseDate ?? m.theatricalReleaseDate,
    })),
    sortField,
    {
      qualityProfileNames: profileNameMap,
      rootFolderNames: rootFolderNameMap,
    },
  )

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

  const allColumns = useMemo(
    () => [...MOVIE_COLUMNS, createMovieActionsColumn({
      onSearch: handleSearch,
      onDelete: handleDelete,
    })],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

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

  const handleBulkMonitor = async (monitored: boolean) => {
    try {
      await bulkUpdateMutation.mutateAsync({
        ids: Array.from(selectedIds),
        data: { monitored },
      })
      toast.success(`${selectedIds.size} movie${selectedIds.size > 1 ? 's' : ''} ${monitored ? 'monitored' : 'unmonitored'}`)
      handleExitEditMode()
    } catch {
      toast.error(`Failed to ${monitored ? 'monitor' : 'unmonitor'} movies`)
    }
  }

  const handleBulkChangeQualityProfile = async (qualityProfileId: number) => {
    try {
      await bulkUpdateMutation.mutateAsync({
        ids: Array.from(selectedIds),
        data: { qualityProfileId },
      })
      const profile = qualityProfiles?.find((p) => p.id === qualityProfileId)
      toast.success(`${selectedIds.size} movie${selectedIds.size > 1 ? 's' : ''} set to "${profile?.name || 'Unknown'}" profile`)
      handleExitEditMode()
    } catch {
      toast.error('Failed to change quality profile')
    }
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
        description={isLoading ? <Skeleton className="h-4 w-36" /> : `${movies?.length || 0} movies in library`}
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={handleRefreshAll}
              disabled={isLoading || refreshAllMutation.isPending || editMode}
            >
              <RefreshCw className={`size-4 mr-1 ${refreshAllMutation.isPending ? 'animate-spin' : ''}`} />
              {refreshAllMutation.isPending ? 'Refreshing...' : 'Refresh'}
            </Button>
            {editMode ? (
              <Button variant="outline" onClick={handleExitEditMode}>
                <X className="size-4 mr-1" />
                Cancel
              </Button>
            ) : (
              <Button variant="outline" onClick={() => setEditMode(true)} disabled={isLoading}>
                <Pencil className="size-4 mr-1" />
                Edit
              </Button>
            )}
            <Button
              disabled={isLoading || editMode}
              className="bg-movie-500 hover:bg-movie-400 border-movie-500"
              onClick={() => document.getElementById('global-search')?.focus()}
            >
              <Plus className="size-4 mr-1" />
              Add Movie
            </Button>
          </div>
        }
      />

      {/* Edit Mode Toolbar */}
      {editMode && (
        <div className="flex items-center gap-4 mb-4 p-3 bg-movie-500/10 border border-movie-500/20 rounded-lg">
          <div className="flex items-center gap-2">
            <Checkbox
              checked={selectedIds.size === filteredMovies.length && filteredMovies.length > 0}
              onCheckedChange={handleSelectAll}
            />
            <span className="text-sm text-muted-foreground">
              {selectedIds.size} of {filteredMovies.length} selected
            </span>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={selectedIds.size === 0 || bulkUpdateMutation.isPending}
              onClick={() => handleBulkMonitor(true)}
            >
              <Eye className="size-4 mr-1" />
              Monitor
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={selectedIds.size === 0 || bulkUpdateMutation.isPending}
              onClick={() => handleBulkMonitor(false)}
            >
              <EyeOff className="size-4 mr-1" />
              Unmonitor
            </Button>
            <Select
              value=""
              onValueChange={(v) => v && handleBulkChangeQualityProfile(Number(v))}
              disabled={selectedIds.size === 0 || bulkUpdateMutation.isPending}
            >
              <SelectTrigger className="w-40 h-8 text-sm">
                Set Quality Profile
              </SelectTrigger>
              <SelectContent>
                {qualityProfiles?.map((profile) => (
                  <SelectItem key={profile.id} value={String(profile.id)}>
                    {profile.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              variant="destructive"
              size="sm"
              disabled={selectedIds.size === 0}
              onClick={() => setShowDeleteDialog(true)}
            >
              <Trash2 className="size-4 mr-1" />
              Delete
            </Button>
          </div>
        </div>
      )}

      {/* Filters & Sort */}
      <div className="flex flex-wrap items-center gap-2 mb-6">
        <FilterDropdown
          options={FILTER_OPTIONS}
          selected={statusFilters}
          onToggle={(v) => setStatusFilters((prev) => prev.includes(v) ? prev.filter((f) => f !== v) : [...prev, v])}
          onReset={() => setStatusFilters([...ALL_FILTERS])}
          label="Statuses"
          theme="movie"
          disabled={isLoading}
        />

        <Select
          value={sortField}
          onValueChange={(v) => v && setSortField(v as SortField)}
          disabled={isLoading}
        >
          <SelectTrigger className="gap-1.5">
            <ArrowUpDown className={cn('size-4 shrink-0', sortField !== 'title' ? 'text-movie-400' : 'text-muted-foreground')} />
            <span className="hidden sm:inline">{SORT_OPTIONS.find((o) => o.value === sortField)?.label}</span>
          </SelectTrigger>
          <SelectContent>
            {SORT_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="ml-auto flex items-center gap-4">
          {moviesView === 'grid' && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Size</span>
              <Slider
                value={[posterSize]}
                onValueChange={(v) => setPosterSize(Array.isArray(v) ? v[0] : v)}
                min={100}
                max={250}
                step={10}
                className="w-24"
                disabled={isLoading}
              />
            </div>
          )}
          {moviesView === 'table' && (
            <ColumnConfigPopover
              columns={MOVIE_COLUMNS}
              visibleColumnIds={movieTableColumns}
              onVisibleColumnsChange={setMovieTableColumns}
              theme="movie"
            />
          )}
          <ToggleGroup
            value={[moviesView]}
            onValueChange={(v) => v.length > 0 && setMoviesView(v[0] as 'grid' | 'table')}
            disabled={isLoading}
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
      {isLoading ? (
        <LoadingState variant={moviesView === 'grid' ? 'card' : 'list'} posterSize={posterSize} theme="movie" />
      ) : sortedMovies.length === 0 ? (
        <EmptyState
          icon={<Film className="size-8 text-movie-500" />}
          title="No movies found"
          description={
            !allFiltersSelected
              ? 'Try adjusting your filters'
              : 'Add your first movie to get started'
          }
          action={
            allFiltersSelected
              ? { label: 'Add Movie', onClick: () => {} }
              : undefined
          }
        />
      ) : moviesView === 'table' ? (
        <MovieTable
          movies={sortedMovies}
          columns={allColumns}
          visibleColumnIds={movieTableColumns}
          renderContext={renderContext}
          sortField={sortField}
          sortDirection={sortDirection}
          onSort={handleColumnSort}
          editMode={editMode}
          selectedIds={selectedIds}
          onToggleSelect={handleToggleSelect}
        />
      ) : groups ? (
        <GroupedMovieGrid
          groups={groups}
          posterSize={posterSize}
          editMode={editMode}
          selectedIds={selectedIds}
          onToggleSelect={handleToggleSelect}
        />
      ) : (
        <MovieGrid
          movies={sortedMovies}
          posterSize={posterSize}
          editMode={editMode}
          selectedIds={selectedIds}
          onToggleSelect={handleToggleSelect}
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
