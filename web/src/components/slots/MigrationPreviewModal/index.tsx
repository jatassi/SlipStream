import { Loader2 } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

import { AssignModal } from '../shared/assign-modal'
import type { DryRunModalProps } from '../shared/types'
import { DialogFooterActions } from './dialog-footer-content'
import { MediaListArea } from './media-list-area'
import { SummaryCardsGrid } from './summary-cards-grid'
import { TabToolbar } from './tab-toolbar'
import { useMigrationPreviewModal } from './use-migration-preview-modal'

export function DryRunModal(props: DryRunModalProps) {
  const { open, onOpenChange } = props
  const state = useMigrationPreviewModal(props)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Migration Dry Run Preview</DialogTitle>
          <DialogDescription>
            Review how your existing files will be assigned to version slots
          </DialogDescription>
        </DialogHeader>

        <ModalBody state={state} />

        <DialogFooter className="mt-2 shrink-0">
          <DialogFooterActions
            developerMode={state.developerMode}
            isDebugData={state.isDebugData}
            isLoadingDebugData={state.isLoadingDebugData}
            isLoading={state.isLoading}
            isExecuting={state.isExecuting}
            hasPreview={!!state.editedPreview}
            onLoadDebugData={state.handleLoadDebugData}
            onCancel={() => onOpenChange(false)}
            onExecute={state.handleExecute}
          />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type ModalBodyProps = {
  state: ReturnType<typeof useMigrationPreviewModal>
}

function ModalBody({ state }: ModalBodyProps) {
  if (state.isLoading) {
    return <LoadingState isDebug={state.isLoadingDebugData} />
  }
  if (!state.editedPreview) {
    return (
      <div className="text-muted-foreground flex items-center justify-center py-16">
        Failed to load preview
      </div>
    )
  }

  const { summary, movies, tvShows } = state.editedPreview
  return (
    <>
      <SummaryCardsGrid summary={summary} filter={state.filter} onFilterChange={state.setFilter} />
      <TabToolbar
        activeTab={state.activeTab}
        onTabChange={state.setActiveTab}
        movieCount={movies.length}
        tvShowCount={tvShows.length}
        allSelected={state.allSelected}
        visibleCount={state.visibleFileIds.length}
        selectedCount={state.selectedCount}
        hasEdits={state.manualEditsCount > 0}
        onToggleSelectAll={state.handleToggleSelectAll}
        onIgnore={state.handleIgnore}
        onOpenAssign={() => state.setAssignModalOpen(true)}
        onUnassign={state.handleUnassign}
        onReset={state.handleReset}
      />
      <ScrollArea className="h-[50vh]">
        <MediaListArea
          activeTab={state.activeTab}
          filteredMovies={state.filteredMovies}
          filteredTvShows={state.filteredTvShows}
          selectedFileIds={state.selectedFileIds}
          ignoredFileIds={state.ignoredFileIds}
          onToggleFileSelection={state.handleToggleFileSelection}
        />
      </ScrollArea>
      <AssignModal
        open={state.assignModalOpen}
        onOpenChange={state.setAssignModalOpen}
        slots={state.enabledSlots}
        selectedCount={state.selectedCount}
        onAssign={state.handleAssign}
      />
    </>
  )
}

function LoadingState({ isDebug }: { isDebug: boolean }) {
  const message = isDebug
    ? 'Generating debug data with real slot matching...'
    : 'Analyzing library files...'

  return (
    <div className="flex items-center justify-center py-16">
      <Loader2 className="text-muted-foreground size-8 animate-spin" />
      <span className="text-muted-foreground ml-3">{message}</span>
    </div>
  )
}
