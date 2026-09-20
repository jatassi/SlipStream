import { ExternalLink, TestTube } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { LoadingButton } from '@/components/ui/loading-button'

import type { NotificationDialogProps } from './notification-dialog-types'
import { NotificationFormBody } from './notification-form-body'
import { useNotificationDialog } from './use-notification-dialog'

export function NotificationDialog(props: NotificationDialogProps) {
  const state = useNotificationDialog(props)

  return (
    <SheetPresenter
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={state.isEditing ? 'Edit Notification' : 'Add Notification'}
      description={
        <Description
          description={state.currentSchema?.description}
          infoUrl={state.currentSchema?.infoUrl}
        />
      }
      wideClassName="sm:max-w-lg"
      footer={
        <NotificationFooter
          isTesting={state.isTesting}
          isPending={state.isPending}
          isEditing={state.isEditing}
          onTest={state.handleTest}
          onSubmit={state.handleSubmit}
          onCancel={() => props.onOpenChange(false)}
        />
      }
    >
      <NotificationFormBody state={state} />
    </SheetPresenter>
  )
}

function Description({ description, infoUrl }: { description?: string; infoUrl?: string }) {
  return (
    <>
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
    </>
  )
}

function NotificationFooter({
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
    <FormActions
      leading={
        <LoadingButton
          className="min-h-tap"
          loading={isTesting}
          icon={TestTube}
          variant="outline"
          onClick={onTest}
        >
          Test
        </LoadingButton>
      }
      onCancel={onCancel}
      confirmLabel={isEditing ? 'Save' : 'Add'}
      onConfirm={onSubmit}
      loading={isPending}
    />
  )
}
