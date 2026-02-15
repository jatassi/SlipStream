import { Loader2 } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { ConfirmModal } from './confirm-modal'
import { MediaContent } from './media-content'
import { ModalFooter } from './modal-footer'
import { SummaryCards } from './summary-cards'
import { TabBar } from './tab-bar'
import type { DryRunModalProps } from './types'
import { useDryRunModal } from './use-dry-run-modal'

export function DryRunModal(props: DryRunModalProps) {
  const s = useDryRunModal(props)

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-4xl">
        <ModalHeader />
        <ModalBody state={s} />
        <ModalFooter
          developerMode={s.developerMode}
          isDebugData={s.isDebugData}
          isLoadingDebugData={s.isLoadingDebugData}
          isLoading={s.isLoading}
          isExecuting={s.isExecuting}
          canEnable={!s.isLoading && !s.isExecuting && !!s.editedPreview && !s.isDebugData && s.allFilesAccountedFor}
          onLoadDebugData={s.handleLoadDebugData}
          onCancel={() => props.onOpenChange(false)}
          onEnable={() => s.setConfirmModalOpen(true)}
        />
      </DialogContent>
      <ConfirmModal
        open={s.confirmModalOpen}
        onOpenChange={s.setConfirmModalOpen}
        editedPreview={s.editedPreview}
        ignoredCount={s.ignoredFileIds.size}
        isExecuting={s.isExecuting}
        onExecute={s.handleExecute}
      />
    </Dialog>
  )
}

function ModalHeader() {
  return (
    <DialogHeader>
      <DialogTitle>Migration Dry Run Preview</DialogTitle>
      <DialogDescription>
        Review how your existing files will be assigned to version slots
      </DialogDescription>
    </DialogHeader>
  )
}

type ModalBodyProps = {
  state: ReturnType<typeof useDryRunModal>
}

function ModalBody({ state }: ModalBodyProps) {
  if (state.isLoading) {
    return <LoadingState isDebug={state.isLoadingDebugData} />
  }
  if (!state.editedPreview) {
    return <ErrorState />
  }
  return (
    <>
      <SummaryCards summary={state.editedPreview.summary} filter={state.filter} onFilterChange={state.setFilter} />
      <TabBar
        activeTab={state.activeTab}
        onTabChange={state.setActiveTab}
        preview={state.editedPreview}
        visibleFileIds={state.visibleFileIds}
        selectedFileIds={state.selectedFileIds}
        manualEditsCount={state.manualEditsCount}
        onToggleSelectAll={state.handleToggleSelectAll}
        onIgnore={state.handleIgnore}
        onOpenAssign={() => state.setAssignModalOpen(true)}
        onUnassign={state.handleUnassign}
        onReset={state.handleReset}
      />
      <MediaContent
        activeTab={state.activeTab}
        filteredMovies={state.filteredMovies}
        filteredTvShows={state.filteredTvShows}
        selectedFileIds={state.selectedFileIds}
        ignoredFileIds={state.ignoredFileIds}
        onToggleFileSelection={state.handleToggleFileSelection}
        assignModalOpen={state.assignModalOpen}
        setAssignModalOpen={state.setAssignModalOpen}
        enabledSlots={state.enabledSlots}
        onAssign={state.handleAssign}
      />
    </>
  )
}

function LoadingState({ isDebug }: { isDebug: boolean }) {
  const message = isDebug ? 'Generating debug data with real slot matching...' : 'Analyzing library files...'
  return (
    <div className="flex items-center justify-center py-16">
      <Loader2 className="text-muted-foreground size-8 animate-spin" />
      <span className="text-muted-foreground ml-3">{message}</span>
    </div>
  )
}

function ErrorState() {
  return (
    <div className="text-muted-foreground flex items-center justify-center py-16">
      Failed to load preview
    </div>
  )
}
