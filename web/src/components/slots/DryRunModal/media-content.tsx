import { ScrollArea } from '@/components/ui/scroll-area'

import { AssignModal } from './assign-modal'
import { MoviesList } from './movies-list'
import { TVShowsList } from './series-list'
import type { MovieMigrationPreview, Slot, TVShowMigrationPreview } from './types'

type MediaContentProps = {
  activeTab: 'movies' | 'tv'
  filteredMovies: MovieMigrationPreview[]
  filteredTvShows: TVShowMigrationPreview[]
  selectedFileIds: Set<number>
  ignoredFileIds: Set<number>
  onToggleFileSelection: (fileId: number) => void
  assignModalOpen: boolean
  setAssignModalOpen: (v: boolean) => void
  enabledSlots: Slot[]
  onAssign: (slotId: number, slotName: string) => void
}

export function MediaContent(props: MediaContentProps) {
  const listContent =
    props.activeTab === 'movies' ? (
      <MoviesList
        movies={props.filteredMovies}
        selectedFileIds={props.selectedFileIds}
        ignoredFileIds={props.ignoredFileIds}
        onToggleFileSelection={props.onToggleFileSelection}
      />
    ) : (
      <TVShowsList
        shows={props.filteredTvShows}
        selectedFileIds={props.selectedFileIds}
        ignoredFileIds={props.ignoredFileIds}
        onToggleFileSelection={props.onToggleFileSelection}
      />
    )

  return (
    <>
      <ScrollArea className="h-[50vh]">{listContent}</ScrollArea>
      <AssignModal
        open={props.assignModalOpen}
        onOpenChange={props.setAssignModalOpen}
        slots={props.enabledSlots}
        selectedCount={props.selectedFileIds.size}
        onAssign={props.onAssign}
      />
    </>
  )
}
