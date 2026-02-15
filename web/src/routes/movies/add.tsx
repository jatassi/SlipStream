import { ArrowLeft, Loader2 } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'

import { AddMovieConfigure } from './add-movie-configure'
import { AddMovieSearch } from './add-movie-search'
import { useAddMoviePage } from './use-add-movie'

export function AddMoviePage() {
  const state = useAddMoviePage()

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

      {state.tmdbId && state.loadingMetadata ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="text-muted-foreground size-8 animate-spin" />
        </div>
      ) : null}

      {state.step === 'search' && !state.tmdbId && (
        <AddMovieSearch
          searchQuery={state.searchQuery}
          onSearchChange={state.setSearchQuery}
          searchInputRef={state.searchInputRef}
          searching={state.searching}
          searchResults={state.searchResults}
          onSelect={state.handleSelectMovie}
        />
      )}

      {state.step === 'configure' && state.selectedMovie ? (
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
