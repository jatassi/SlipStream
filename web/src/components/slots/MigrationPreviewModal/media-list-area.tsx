import { MoviesList } from './movies-list'
import { TVShowsList } from './series-list'
import type { MovieMigrationPreview, TVShowMigrationPreview } from './types'

type MediaListAreaProps = {
  activeTab: 'movies' | 'tv'
  filteredMovies: MovieMigrationPreview[]
  filteredTvShows: TVShowMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
}

export function MediaListArea({
  activeTab,
  filteredMovies,
  filteredTvShows,
  selectedFileIds,
  ignoredFileIds,
  onToggleFileSelection,
}: MediaListAreaProps) {
  if (activeTab === 'movies') {
    return (
      <MoviesList
        movies={filteredMovies}
        selectedFileIds={selectedFileIds}
        ignoredFileIds={ignoredFileIds}
        onToggleFileSelection={onToggleFileSelection}
      />
    )
  }

  return (
    <TVShowsList
      shows={filteredTvShows}
      selectedFileIds={selectedFileIds}
      ignoredFileIds={ignoredFileIds}
      onToggleFileSelection={onToggleFileSelection}
    />
  )
}
