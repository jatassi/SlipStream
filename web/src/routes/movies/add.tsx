import { useEffect } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'

import { AddMovieConfigure } from './add-movie-configure'
import { useAddMoviePage } from './use-add-movie'

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

      {state.loadingMetadata && !state.selectedMovie ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="text-muted-foreground size-8 animate-spin" />
        </div>
      ) : null}

      {state.selectedMovie ? (
        <AddMovieConfigure
          selectedMovie={state.selectedMovie}
          rootFolderId={state.rootFolderId}
          setRootFolderId={state.setRootFolderId}
          rootFolders={state.rootFolders}
          qualityProfileId={state.qualityProfileId}
          setQualityProfileId={state.setQualityProfileId}
          qualityProfiles={state.qualityProfiles}
          monitored={state.monitored}
          setMonitored={state.setMonitored}
          searchOnAdd={state.searchOnAdd}
          setSearchOnAdd={state.setSearchOnAdd}
          isPending={state.isPending}
          handleBack={state.handleBack}
          handleAdd={state.handleAdd}
        />
      ) : null}
    </div>
  )
}
