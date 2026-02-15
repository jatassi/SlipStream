import { ExternalLink, Loader2, TestTube } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import type { NotificationDialogProps } from './notification-dialog-types'
import { NotificationFormBody } from './notification-form-body'
import { useNotificationDialog } from './use-notification-dialog'

export function NotificationDialog(props: NotificationDialogProps) {
  const state = useNotificationDialog(props)

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
        <NotificationDialogHeader
          isEditing={state.isEditing}
          description={state.currentSchema?.description}
          infoUrl={state.currentSchema?.infoUrl}
        />
        <NotificationFormBody state={state} />
        <NotificationDialogFooter
          isTesting={state.isTesting}
          isPending={state.isPending}
          isEditing={state.isEditing}
          onTest={state.handleTest}
          onSubmit={state.handleSubmit}
          onCancel={() => props.onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  )
}

function NotificationDialogHeader({
  isEditing,
  description,
  infoUrl,
}: {
  isEditing: boolean
  description?: string
  infoUrl?: string
}) {
  return (
    <DialogHeader>
      <DialogTitle>{isEditing ? 'Edit Notification' : 'Add Notification'}</DialogTitle>
      <DialogDescription>
        {description ?? 'Configure notification settings and triggers.'}
        {infoUrl ? (
          <a
            href={infoUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary ml-1 inline-flex items-center gap-1 hover:underline"
          >
            Learn more <ExternalLink className="size-3" />
          </a>
        ) : null}
      </DialogDescription>
    </DialogHeader>
  )
}

function NotificationDialogFooter({
  isTesting,
  isPending,
  isEditing,
  onTest,
  onSubmit,
  onCancel,
}: {
  isTesting: boolean
  isPending: boolean
  isEditing: boolean
  onTest: () => void
  onSubmit: () => void
  onCancel: () => void
}) {
  return (
    <DialogFooter className="flex-col gap-2 sm:flex-row">
      <Button variant="outline" onClick={onTest} disabled={isTesting}>
        {isTesting ? (
          <Loader2 className="mr-2 size-4 animate-spin" />
        ) : (
          <TestTube className="mr-2 size-4" />
        )}
        Test
      </Button>
      <div className="flex gap-2 sm:ml-auto">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={onSubmit} disabled={isPending}>
          {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
          {isEditing ? 'Save' : 'Add'}
        </Button>
      </div>
    </DialogFooter>
  )
}
