import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Plus, Grid, List, Tv, RefreshCw } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { SeriesGrid } from '@/components/series/SeriesGrid'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { useSeries, useScanLibrary } from '@/hooks'
import { useUIStore } from '@/stores'
import { toast } from 'sonner'
import type { Series } from '@/types'

type FilterStatus = 'all' | 'monitored' | 'continuing' | 'ended'

export function SeriesListPage() {
  const { seriesView, setSeriesView } = useUIStore()
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')

  const { data: seriesList, isLoading, isError, refetch } = useSeries()
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
              disabled={scanMutation.isPending}
            >
              <RefreshCw className={`size-4 mr-1 ${scanMutation.isPending ? 'animate-spin' : ''}`} />
              {scanMutation.isPending ? 'Scanning...' : 'Refresh'}
            </Button>
            <Link to="/series/add">
              <Button>
                <Plus className="size-4 mr-1" />
                Add Series
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
            <SelectItem value="continuing">Continuing</SelectItem>
            <SelectItem value="ended">Ended</SelectItem>
          </SelectContent>
        </Select>

        <div className="ml-auto">
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
          icon={<Tv className="size-8" />}
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
        <SeriesGrid series={filteredSeries} />
      )}
    </div>
  )
}
