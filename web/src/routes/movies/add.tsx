import { useEffect } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { AddMediaConfigure, MediaPreview } from '@/components/media/add-media-configure'
import { ToggleField } from '@/components/media/media-configure-fields'
import { Button } from '@/components/ui/button'

import { useAddMoviePage } from './use-add-movie'

type PageState = ReturnType<typeof useAddMoviePage>

export function AddMoviePage() {
  const state = useAddMoviePage()
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
    <div>
      <PageHeader
        title="Add Movie"
        breadcrumbs={[{ label: 'Movies', href: '/movies' }, { label: 'Add' }]}
        actions={
          <Button variant="ghost" onClick={state.handleBack}>
            <ArrowLeft className="mr-2 size-4" />
            Back
          </Button>
        }
      />
      <AddMovieBody state={state} />
    </div>
  )
}

function AddMovieBody({ state }: { state: PageState }) {
  if (state.loadingMetadata && !state.selectedMovie) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="text-muted-foreground size-8 animate-spin" />
      </div>
    )
  }

  if (!state.selectedMovie) {
    return null
  }

  return (
    <AddMediaConfigure
      preview={
        <MediaPreview
          title={state.selectedMovie.title}
          year={state.selectedMovie.year}
          overview={state.selectedMovie.overview}
          posterUrl={state.selectedMovie.posterUrl}
          type="movie"
        />
      }
      rootFolders={state.rootFolders}
      qualityProfiles={state.qualityProfiles}
      rootFolderId={state.form.watch('rootFolderId')}
      qualityProfileId={state.form.watch('qualityProfileId')}
      onFolderChange={(v) => state.form.setValue('rootFolderId', v)}
      onProfileChange={(v) => state.form.setValue('qualityProfileId', v)}
      isPending={state.isPending}
      onBack={state.handleBack}
      onAdd={state.handleAdd}
      addLabel="Add Movie"
    >
      <ToggleField
        label="Monitored"
        description="Automatically search for and download releases"
        checked={state.form.watch('monitored')}
        onChange={(v) => state.form.setValue('monitored', v)}
      />
      <ToggleField
        label="Search on Add"
        description="Start searching for releases immediately"
        checked={state.form.watch('searchOnAdd')}
        onChange={(v) => state.form.setValue('searchOnAdd', v)}
      />
    </AddMediaConfigure>
  )
}
