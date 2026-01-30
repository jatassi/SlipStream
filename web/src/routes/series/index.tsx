import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Plus, Grid, List, Tv, RefreshCw, Pencil, Trash2, X, Eye, EyeOff } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
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
import { SeriesGrid } from '@/components/series/SeriesGrid'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { useSeries, useBulkDeleteSeries, useBulkUpdateSeries, useScanLibrary, useQualityProfiles } from '@/hooks'
import { useUIStore } from '@/stores'
import { toast } from 'sonner'
import type { Series } from '@/types'

type FilterStatus = 'all' | 'monitored' | 'continuing' | 'ended'

export function SeriesListPage() {
  const { seriesView, setSeriesView, posterSize, setPosterSize } = useUIStore()
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')
  const [editMode, setEditMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)

  const { data: seriesList, isLoading, isError, refetch } = useSeries()
  const { data: qualityProfiles } = useQualityProfiles()
  const bulkDeleteMutation = useBulkDeleteSeries()
  const bulkUpdateMutation = useBulkUpdateSeries()
  const scanMutation = useScanLibrary()

  const handleScanLibrary = async () => {
    try {
      await scanMutation.mutateAsync()
      toast.success('Library scan started')
    } catch {
      toast.error('Failed to start library scan')
    }
  }

  // Filter series by status
  const filteredSeries = (seriesList || []).filter((s: Series) => {
    if (statusFilter === 'monitored' && !s.monitored) return false
    if (statusFilter === 'continuing' && s.status !== 'continuing') return false
    if (statusFilter === 'ended' && s.status !== 'ended') return false
    return true
  })

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
    if (selectedIds.size === filteredSeries.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredSeries.map((s) => s.id)))
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
      toast.success(`${selectedIds.size} series deleted`)
      setShowDeleteDialog(false)
      setDeleteFiles(false)
      handleExitEditMode()
    } catch {
      toast.error('Failed to delete series')
    }
  }

  const handleBulkMonitor = async (monitored: boolean) => {
    try {
      await bulkUpdateMutation.mutateAsync({
        ids: Array.from(selectedIds),
        data: { monitored },
      })
      toast.success(`${selectedIds.size} series ${monitored ? 'monitored' : 'unmonitored'}`)
      handleExitEditMode()
    } catch {
      toast.error(`Failed to ${monitored ? 'monitor' : 'unmonitor'} series`)
    }
  }

  const handleBulkChangeQualityProfile = async (qualityProfileId: number) => {
    try {
      await bulkUpdateMutation.mutateAsync({
        ids: Array.from(selectedIds),
        data: { qualityProfileId },
      })
      const profile = qualityProfiles?.find((p) => p.id === qualityProfileId)
      toast.success(`${selectedIds.size} series set to "${profile?.name || 'Unknown'}" profile`)
      handleExitEditMode()
    } catch {
      toast.error('Failed to change quality profile')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Series" />
        <LoadingState variant="card" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Series" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Series"
        description={`${seriesList?.length || 0} series in library`}
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
            <Link to="/series/add">
              <Button disabled={editMode} className="bg-tv-500 hover:bg-tv-400 border-tv-500">
                <Plus className="size-4 mr-1" />
                Add Series
              </Button>
            </Link>
          </div>
        }
      />

      {/* Edit Mode Toolbar */}
      {editMode && (
        <div className="flex items-center gap-4 mb-4 p-3 bg-tv-500/10 border border-tv-500/20 rounded-lg">
          <div className="flex items-center gap-2">
            <Checkbox
              checked={selectedIds.size === filteredSeries.length && filteredSeries.length > 0}
              onCheckedChange={handleSelectAll}
            />
            <span className="text-sm text-muted-foreground">
              {selectedIds.size} of {filteredSeries.length} selected
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

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <Select
          value={statusFilter}
          onValueChange={(v) => v && setStatusFilter(v as FilterStatus)}
        >
          <SelectTrigger className="w-36">
            {{ all: 'All', monitored: 'Monitored', continuing: 'Continuing', ended: 'Ended' }[statusFilter]}
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="monitored">Monitored</SelectItem>
            <SelectItem value="continuing">Continuing</SelectItem>
            <SelectItem value="ended">Ended</SelectItem>
          </SelectContent>
        </Select>

        <div className="ml-auto flex items-center gap-4">
          {seriesView === 'grid' && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Size</span>
              <Slider
                value={[posterSize]}
                onValueChange={(v) => setPosterSize(Array.isArray(v) ? v[0] : v)}
                min={100}
                max={250}
                step={10}
                className="w-24"
              />
            </div>
          )}
          <ToggleGroup
            value={[seriesView]}
            onValueChange={(v) => v.length > 0 && setSeriesView(v[0] as 'grid' | 'table')}
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
      {filteredSeries.length === 0 ? (
        <EmptyState
          icon={<Tv className="size-8 text-tv-500" />}
          title="No series found"
          description={
            statusFilter !== 'all'
              ? 'Try adjusting your filters'
              : 'Add your first series to get started'
          }
          action={
            statusFilter === 'all'
              ? { label: 'Add Series', onClick: () => {} }
              : undefined
          }
        />
      ) : (
        <SeriesGrid
          series={filteredSeries}
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
            <AlertDialogTitle>Delete {selectedIds.size} Series?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. The selected series will be removed from your library.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 py-2">
            <Checkbox
              id="deleteSeriesFiles"
              checked={deleteFiles}
              onCheckedChange={(checked) => setDeleteFiles(checked === true)}
            />
            <label htmlFor="deleteSeriesFiles" className="text-sm cursor-pointer">
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
