import { useEffect } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'

import {
  AddMediaActions,
  AddMediaConfigure,
  MediaPreview,
} from '@/components/media/add-media-configure'
import { AddMediaPresenter } from '@/components/media/add-media-presenter'
import { MonitorSelect, SearchOnAddSelect, ToggleField } from '@/components/media/media-configure-fields'

import { useAddSeriesPage } from './use-add-series'

type PageState = ReturnType<typeof useAddSeriesPage>

export function AddSeriesPage() {
  const state = useAddSeriesPage()
  const navigate = useNavigate()

  useEffect(() => {
    if (!state.tmdbId) {
      void navigate({ to: '/search', search: { q: '' } })
    }
  }, [state.tmdbId, navigate])

  if (!state.tmdbId) {
    return <div />
  }

  return (
    <AddMediaPresenter
      title="Add Series"
      backLabel="Library"
      onClose={state.handleBack}
      actions={
        <AddMediaActions
          rootFolderId={state.form.watch('rootFolderId')}
          qualityProfileId={state.form.watch('qualityProfileId')}
          isPending={state.isPending}
          onCancel={state.handleBack}
          onAdd={state.handleAdd}
          addLabel="Add Series"
        />
      }
    >
      <AddSeriesBody state={state} />
    </AddMediaPresenter>
  )
}

function AddSeriesBody({ state }: { state: PageState }) {
  if (state.loadingMetadata && !state.selectedSeries) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  if (!state.selectedSeries) {
    return null
  }

  return <SeriesConfigure state={{ ...state, selectedSeries: state.selectedSeries }} />
}

function SeriesConfigure({ state }: { state: PageState & { selectedSeries: NonNullable<PageState['selectedSeries']> } }) {
  const { form } = state
  return (
    <AddMediaConfigure
      preview={
        <MediaPreview
          title={state.selectedSeries.title}
          year={state.selectedSeries.year}
          overview={state.selectedSeries.overview}
          posterUrl={state.selectedSeries.posterUrl}
          type="series"
          subtitle={state.selectedSeries.network}
        />
      }
      rootFolders={state.rootFolders}
      qualityProfiles={state.qualityProfiles}
      rootFolderId={form.watch('rootFolderId')}
      qualityProfileId={form.watch('qualityProfileId')}
      onFolderChange={(v) => form.setValue('rootFolderId', v)}
      onProfileChange={(v) => form.setValue('qualityProfileId', v)}
    >
      <MonitorSelect value={form.watch('monitorOnAdd')} onChange={(v) => form.setValue('monitorOnAdd', v)} />
      <SearchOnAddSelect value={form.watch('searchOnAdd')} onChange={(v) => form.setValue('searchOnAdd', v)} />
      <ToggleField label="Season Folder" description="Organize episodes into season folders" checked={form.watch('seasonFolder')} onChange={(v) => form.setValue('seasonFolder', v)} />
      <ToggleField label="Include Specials" description="Monitor and search for special episodes (Season 0)" checked={form.watch('includeSpecials')} onChange={(v) => form.setValue('includeSpecials', v)} />
    </AddMediaConfigure>
  )
}
