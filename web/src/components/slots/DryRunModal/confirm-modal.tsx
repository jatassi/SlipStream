import { Layers } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import type { MigrationPreview } from '../shared/types'

type ConfirmModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  editedPreview: MigrationPreview | null
  ignoredCount: number
  isExecuting: boolean
  onExecute: () => void
}

export function ConfirmModal(props: ConfirmModalProps) {
  const { open, onOpenChange, editedPreview, ignoredCount, isExecuting, onExecute } = props

  return (
    <SheetPresenter
      nested
      open={open}
      onOpenChange={onOpenChange}
      title="Enable Multi-Version Mode"
      description="You are about to enable multi-version mode for your library."
      wideClassName="sm:max-w-lg"
      footer={
        <FormActions
          cancelLabel="Back"
          onCancel={() => onOpenChange(false)}
          cancelDisabled={isExecuting}
          confirmLabel={isExecuting ? 'Enabling...' : 'Proceed'}
          onConfirm={onExecute}
          loading={isExecuting}
        />
      }
    >
      <div className="space-y-4 py-2">
        <MigrationInfoAlert />
        {editedPreview !== null && (
          <ConfirmStats summary={editedPreview.summary} ignoredCount={ignoredCount} />
        )}
      </div>
    </SheetPresenter>
  )
}

function MigrationInfoAlert() {
  return (
    <Alert className="border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/50">
      <Layers className="size-4 text-blue-600 dark:text-blue-400" />
      <AlertTitle className="text-blue-800 dark:text-blue-200">What will happen:</AlertTitle>
      <AlertDescription className="text-blue-700 dark:text-blue-300">
        <ul className="mt-2 list-inside list-disc space-y-1">
          <li>Your existing files will be assigned to version slots based on the preview</li>
          <li>Files will be organized according to your slot configuration</li>
          <li>Future downloads will automatically be routed to the appropriate slot</li>
          <li>You can disable multi-version mode later, but file assignments will be preserved</li>
        </ul>
      </AlertDescription>
    </Alert>
  )
}

function ConfirmStats({ summary, ignoredCount }: { summary: MigrationPreview['summary']; ignoredCount: number }) {
  return (
    <div className="grid grid-cols-3 gap-3 text-center">
      <div className="bg-muted/50 rounded-lg p-3">
        <div className="text-2xl font-bold">{summary.totalFiles}</div>
        <div className="text-muted-foreground text-xs">Total Files</div>
      </div>
      <div className="rounded-lg bg-green-50 p-3 dark:bg-green-950/50">
        <div className="text-2xl font-bold text-green-600 dark:text-green-400">{summary.filesWithSlots}</div>
        <div className="text-muted-foreground text-xs">Will Be Assigned</div>
      </div>
      <div className="rounded-lg bg-orange-50 p-3 dark:bg-orange-950/50">
        <div className="text-2xl font-bold text-orange-600 dark:text-orange-400">{ignoredCount}</div>
        <div className="text-muted-foreground text-xs">Ignored</div>
      </div>
    </div>
  )
}
