import { ArrowLeft, Loader2 } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'

import { AddSeriesConfigure } from './add-series-configure'
import { AddSeriesSearch } from './add-series-search'
import type { AddSeriesState } from './use-add-series'
import { useAddSeriesPage } from './use-add-series'

const ADD_BREADCRUMBS = [{ label: 'Series', href: '/series' }, { label: 'Add' }]

function AddSeriesBody({ state }: { state: AddSeriesState }) {
  if (state.tmdbId && state.loadingMetadata) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }
  if (state.step === 'search' && !state.tmdbId) {
    return (
      <AddSeriesSearch
        searchQuery={state.searchQuery} onSearchChange={state.setSearchQuery} searchInputRef={state.searchInputRef}
        searching={state.searching} searchResults={state.searchResults} onSelect={state.handleSelectSeries}
      />
    )
  }
  if (state.step === 'configure' && state.selectedSeries) {
    return (
      <AddSeriesConfigure
        selectedSeries={state.selectedSeries} rootFolderId={state.rootFolderId} setRootFolderId={state.setRootFolderId}
        rootFolders={state.rootFolders} qualityProfileId={state.qualityProfileId} setQualityProfileId={state.setQualityProfileId}
        qualityProfiles={state.qualityProfiles} monitorOnAdd={state.monitorOnAdd} setMonitorOnAdd={state.setMonitorOnAdd}
        searchOnAdd={state.searchOnAdd} setSearchOnAdd={state.setSearchOnAdd} seasonFolder={state.seasonFolder}
        setSeasonFolder={state.setSeasonFolder} includeSpecials={state.includeSpecials} setIncludeSpecials={state.setIncludeSpecials}
        isPending={state.isPending} handleBack={state.handleBack} handleAdd={state.handleAdd}
      />
    )
  }
  return null
}

export function AddSeriesPage() {
  const state = useAddSeriesPage()
  return (
    <div>
      <PageHeader
        title="Add Series"
        breadcrumbs={ADD_BREADCRUMBS}
        actions={<Button variant="ghost" onClick={state.handleBack}><ArrowLeft className="mr-2 size-4" />Back</Button>}
      />
      <AddSeriesBody state={state} />
    </div>
  )
}
