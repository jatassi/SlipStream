import { useMemo, useState } from 'react'

import { toast } from 'sonner'

import {
  useBulkDeleteMovies,
  useBulkMonitorMovies,
  useBulkUpdateMovies,
  useDeleteMovie,
  useGlobalLoading,
  useMovies,
  useQualityProfiles,
  useRefreshAllMovies,
  useRootFolders,
  useSearchMovie,
} from '@/hooks'
import { groupMedia } from '@/lib/grouping'
import type { ColumnRenderContext } from '@/lib/table-columns'
import {
  createMovieActionsColumn,
  DEFAULT_SORT_DIRECTIONS,
  MOVIE_COLUMNS,
} from '@/lib/table-columns'
import { useUIStore } from '@/stores'

import { filterMovies, sortMovies } from './movie-list-utils'

export type FilterStatus =
  | 'monitored'
  | 'unreleased'
  | 'missing'
  | 'downloading'
  | 'failed'
  | 'upgradable'
  | 'available'

export type SortField =
  | 'title'
  | 'monitored'
  | 'qualityProfile'
  | 'releaseDate'
  | 'dateAdded'
  | 'rootFolder'
  | 'sizeOnDisk'

const ALL_FILTERS: FilterStatus[] = [
  'monitored',
  'unreleased',
  'missing',
  'downloading',
  'failed',
  'upgradable',
  'available',
]

export type MovieListState = ReturnType<typeof useMovieList>

export function useMovieList() {
  const ui = useUIState()
  const local = useLocalState()
  const queries = useQueryLayer()
  const derived = useDerivedData(local, queries)
  const edit = useEditHandlers({ local, derived })
  const bulk = useBulkHandlers({ local, queries, onExitEditMode: edit.handleExitEditMode })
  const view = useViewHandlers(local, ui)

  return { ...ui, ...local, ...queries, ...derived, ...edit, ...bulk, ...view }
}

function useUIState() {
  const {
    moviesView,
    setMoviesView,
    posterSize,
    setPosterSize,
    movieTableColumns,
    setMovieTableColumns,
  } = useUIStore()

  return {
    moviesView, setMoviesView,
    posterSize, setPosterSize,
    movieTableColumns, setMovieTableColumns,
  }
}

function useLocalState() {
  const [statusFilters, setStatusFilters] = useState<FilterStatus[]>([...ALL_FILTERS])
  const [sortField, setSortField] = useState<SortField>('title')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [editMode, setEditMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)

  return {
    statusFilters, setStatusFilters,
    sortField, setSortField,
    sortDirection, setSortDirection,
    editMode, setEditMode,
    selectedIds, setSelectedIds,
    showDeleteDialog, setShowDeleteDialog,
    deleteFiles, setDeleteFiles,
  }
}

function useQueryLayer() {
  const globalLoading = useGlobalLoading()
  const { data: movies, isLoading: queryLoading, isError, refetch } = useMovies()
  const isLoading = queryLoading || globalLoading
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: rootFolders } = useRootFolders()
  const searchMutation = useSearchMovie()
  const deleteMutation = useDeleteMovie()
  const bulkDeleteMutation = useBulkDeleteMovies()
  const bulkUpdateMutation = useBulkUpdateMovies()
  const bulkMonitorMutation = useBulkMonitorMovies()
  const refreshAllMutation = useRefreshAllMovies()

  return {
    movies, isLoading, isError, refetch,
    qualityProfiles, rootFolders,
    searchMutation, deleteMutation,
    bulkDeleteMutation, bulkUpdateMutation, bulkMonitorMutation, refreshAllMutation,
  }
}

type LocalState = ReturnType<typeof useLocalState>
type QueryLayer = ReturnType<typeof useQueryLayer>
type UIState = ReturnType<typeof useUIState>

function useDerivedData(local: LocalState, queries: QueryLayer) {
  const profileNameMap = new Map(queries.qualityProfiles?.map((p) => [p.id, p.name]))
  const rootFolderNameMap = new Map(queries.rootFolders?.map((f) => [f.id, f.name]))
  const renderContext: ColumnRenderContext = {
    qualityProfileNames: profileNameMap,
    rootFolderNames: rootFolderNameMap,
  }

  const allFiltersSelected = local.statusFilters.length >= ALL_FILTERS.length
  const filteredMovies = filterMovies(queries.movies ?? [], local.statusFilters, allFiltersSelected)
  const sortedMovies = sortMovies(filteredMovies, {
    sortField: local.sortField,
    sortDirection: local.sortDirection,
    profileNameMap,
  })

  const groups = groupMedia(
    sortedMovies.map((m) => ({
      ...m,
      releaseDate: m.releaseDate ?? m.physicalReleaseDate ?? m.theatricalReleaseDate,
    })),
    local.sortField,
    { qualityProfileNames: profileNameMap, rootFolderNames: rootFolderNameMap },
  )

  const handleSearch = (id: number) => {
    queries.searchMutation.mutate(id, {
      onSuccess: () => toast.success('Search started'),
      onError: () => toast.error('Failed to start search'),
    })
  }

  const handleDelete = (id: number) => {
    queries.deleteMutation.mutate(
      { id },
      {
        onSuccess: () => toast.success('Movie deleted'),
        onError: () => toast.error('Failed to delete movie'),
      },
    )
  }

  const allColumns = useMemo(
    () => [
      ...MOVIE_COLUMNS,
      createMovieActionsColumn({ onSearch: handleSearch, onDelete: handleDelete }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  )

  return {
    profileNameMap, renderContext,
    allFiltersSelected, filteredMovies, sortedMovies, groups, allColumns,
  }
}

type DerivedData = ReturnType<typeof useDerivedData>

type EditDeps = {
  local: LocalState
  derived: DerivedData
}

function useEditHandlers({ local, derived }: EditDeps) {
  const handleExitEditMode = () => {
    local.setEditMode(false)
    local.setSelectedIds(new Set())
  }

  const handleToggleSelect = (id: number) => {
    local.setSelectedIds((prev) => {
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
    if (local.selectedIds.size === derived.filteredMovies.length) {
      local.setSelectedIds(new Set())
    } else {
      local.setSelectedIds(new Set(derived.filteredMovies.map((m) => m.id)))
    }
  }

  return { handleExitEditMode, handleToggleSelect, handleSelectAll }
}

type BulkDeps = {
  local: LocalState
  queries: QueryLayer
  onExitEditMode: () => void
}

function movieLabel(count: number): string {
  return `${count} movie${count > 1 ? 's' : ''}`
}

function useBulkHandlers({ local, queries, onExitEditMode }: BulkDeps) {
  const selectedArray = () => [...local.selectedIds]

  const handleRefreshAll = () => {
    queries.refreshAllMutation.mutate(undefined, {
      onSuccess: () => toast.success('Refresh started for all movies'),
      onError: () => toast.error('Failed to start refresh'),
    })
  }

  const handleBulkDelete = () => {
    queries.bulkDeleteMutation.mutate(
      { ids: selectedArray(), deleteFiles: local.deleteFiles },
      {
        onSuccess: () => {
          toast.success(`${movieLabel(local.selectedIds.size)} deleted`)
          local.setShowDeleteDialog(false)
          local.setDeleteFiles(false)
          onExitEditMode()
        },
        onError: () => toast.error('Failed to delete movies'),
      },
    )
  }

  const handleBulkMonitor = (monitored: boolean) => {
    queries.bulkMonitorMutation.mutate(
      { ids: selectedArray(), monitored },
      {
        onSuccess: () => {
          toast.success(`${movieLabel(local.selectedIds.size)} ${monitored ? 'monitored' : 'unmonitored'}`)
          onExitEditMode()
        },
        onError: () => toast.error(`Failed to ${monitored ? 'monitor' : 'unmonitor'} movies`),
      },
    )
  }

  const handleBulkChangeQualityProfile = (qualityProfileId: number) => {
    queries.bulkUpdateMutation.mutate(
      { ids: selectedArray(), data: { qualityProfileId } },
      {
        onSuccess: () => {
          const name = queries.qualityProfiles?.find((p) => p.id === qualityProfileId)?.name ?? 'Unknown'
          toast.success(`${movieLabel(local.selectedIds.size)} set to "${name}" profile`)
          onExitEditMode()
        },
        onError: () => toast.error('Failed to change quality profile'),
      },
    )
  }

  return { handleRefreshAll, handleBulkDelete, handleBulkMonitor, handleBulkChangeQualityProfile }
}

function useViewHandlers(local: LocalState, ui: UIState) {
  const handleColumnSort = (field: string) => {
    if (field === local.sortField) {
      local.setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      local.setSortField(field as SortField)
      local.setSortDirection(DEFAULT_SORT_DIRECTIONS[field])
    }
  }

  const handleToggleFilter = (v: FilterStatus) => {
    local.setStatusFilters((prev) =>
      prev.includes(v) ? prev.filter((f) => f !== v) : [...prev, v],
    )
  }

  const handleResetFilters = () => local.setStatusFilters([...ALL_FILTERS])

  const handleSortFieldChange = (v: string) => {
    if (v) {
      local.setSortField(v as SortField)
    }
  }

  const handleViewChange = (v: string[]) => {
    if (v.length > 0) {
      ui.setMoviesView(v[0] as 'grid' | 'table')
    }
  }

  const handlePosterSizeChange = (v: number | readonly number[]) => {
    if (Array.isArray(v) && typeof v[0] === 'number') {
      ui.setPosterSize(v[0])
    }
  }

  return {
    handleColumnSort,
    handleToggleFilter,
    handleResetFilters,
    handleSortFieldChange,
    handleViewChange,
    handlePosterSizeChange,
  }
}
