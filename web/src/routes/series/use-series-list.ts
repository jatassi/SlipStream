import { useMemo, useState } from 'react'

import { toast } from 'sonner'

import {
  useBulkDeleteSeries,
  useBulkMonitorSeries,
  useBulkUpdateSeries,
  useGlobalLoading,
  useQualityProfiles,
  useRefreshAllSeries,
  useRootFolders,
  useSeries,
} from '@/hooks'
import { groupMedia } from '@/lib/grouping'
import type { ColumnRenderContext } from '@/lib/table-columns'
import {
  createSeriesActionsColumn,
  DEFAULT_SORT_DIRECTIONS,
  SERIES_COLUMNS,
} from '@/lib/table-columns'
import { useUIStore } from '@/stores'

import { filterSeries, sortSeries } from './series-list-utils'

export type FilterStatus =
  | 'monitored'
  | 'continuing'
  | 'ended'
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
  | 'nextAirDate'
  | 'dateAdded'
  | 'rootFolder'
  | 'sizeOnDisk'

const ALL_FILTERS: FilterStatus[] = [
  'monitored',
  'continuing',
  'ended',
  'unreleased',
  'missing',
  'downloading',
  'failed',
  'upgradable',
  'available',
]

export type SeriesListState = ReturnType<typeof useSeriesList>

export function useSeriesList() {
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
    seriesView,
    setSeriesView,
    posterSize,
    setPosterSize,
    seriesTableColumns,
    setSeriesTableColumns,
  } = useUIStore()

  return {
    seriesView, setSeriesView,
    posterSize, setPosterSize,
    seriesTableColumns, setSeriesTableColumns,
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
  const { data: seriesList, isLoading: queryLoading, isError, refetch } = useSeries()
  const isLoading = queryLoading || globalLoading
  const { data: qualityProfiles } = useQualityProfiles()
  const { data: rootFolders } = useRootFolders()
  const bulkDeleteMutation = useBulkDeleteSeries()
  const bulkUpdateMutation = useBulkUpdateSeries()
  const bulkMonitorMutation = useBulkMonitorSeries()
  const refreshAllMutation = useRefreshAllSeries()

  return {
    seriesList, isLoading, isError, refetch,
    qualityProfiles, rootFolders,
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
  const filteredSeries = filterSeries(queries.seriesList ?? [], local.statusFilters, allFiltersSelected)
  const sortedSeries = sortSeries(filteredSeries, {
    sortField: local.sortField,
    sortDirection: local.sortDirection,
    profileNameMap,
  })

  const groups = groupMedia(sortedSeries, local.sortField, {
    qualityProfileNames: profileNameMap,
    rootFolderNames: rootFolderNameMap,
  })

  const allColumns = useMemo(
    () => [...SERIES_COLUMNS, createSeriesActionsColumn({})],
    [],
  )

  return {
    profileNameMap, renderContext,
    allFiltersSelected, filteredSeries, sortedSeries, groups, allColumns,
  }
}

type EditDeps = {
  local: LocalState
  derived: ReturnType<typeof useDerivedData>
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
    if (local.selectedIds.size === derived.filteredSeries.length) {
      local.setSelectedIds(new Set())
    } else {
      local.setSelectedIds(new Set(derived.filteredSeries.map((s) => s.id)))
    }
  }

  return { handleExitEditMode, handleToggleSelect, handleSelectAll }
}

type BulkDeps = {
  local: LocalState
  queries: QueryLayer
  onExitEditMode: () => void
}

function useBulkHandlers({ local, queries, onExitEditMode }: BulkDeps) {
  const selectedArray = () => [...local.selectedIds]

  const handleRefreshAll = () => {
    queries.refreshAllMutation.mutate(undefined, {
      onSuccess: () => toast.success('Refresh started for all series'),
      onError: () => toast.error('Failed to start refresh'),
    })
  }

  const handleBulkDelete = () => {
    queries.bulkDeleteMutation.mutate(
      { ids: selectedArray(), deleteFiles: local.deleteFiles },
      {
        onSuccess: () => {
          toast.success(`${local.selectedIds.size} series deleted`)
          local.setShowDeleteDialog(false)
          local.setDeleteFiles(false)
          onExitEditMode()
        },
        onError: () => toast.error('Failed to delete series'),
      },
    )
  }

  const handleBulkMonitor = (monitored: boolean) => {
    queries.bulkMonitorMutation.mutate(
      { ids: selectedArray(), monitored },
      {
        onSuccess: () => {
          toast.success(`${local.selectedIds.size} series ${monitored ? 'monitored' : 'unmonitored'}`)
          onExitEditMode()
        },
        onError: () => toast.error(`Failed to ${monitored ? 'monitor' : 'unmonitor'} series`),
      },
    )
  }

  const handleBulkChangeQualityProfile = (qualityProfileId: number) => {
    queries.bulkUpdateMutation.mutate(
      { ids: selectedArray(), data: { qualityProfileId } },
      {
        onSuccess: () => {
          const name = queries.qualityProfiles?.find((p) => p.id === qualityProfileId)?.name ?? 'Unknown'
          toast.success(`${local.selectedIds.size} series set to "${name}" profile`)
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
      ui.setSeriesView(v[0] as 'grid' | 'table')
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
