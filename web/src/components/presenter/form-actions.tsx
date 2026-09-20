import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { LoadingButton } from '@/components/ui/loading-button'

export type FormActionsProps = {
  /** Secondary actions (Test, Debug) that sit before the cancel and confirm pair. */
  leading?: ReactNode
  cancelLabel?: string
  onCancel: () => void
  cancelDisabled?: boolean
  confirmLabel: ReactNode
  /** Variant for the confirm button; `destructive` for a confirm that denies or deletes. */
  confirmVariant?: 'default' | 'destructive'
  onConfirm: () => void
  confirmDisabled?: boolean
  loading?: boolean
}

/** The footer every presented form shares: 44 px buttons, stacked on a phone. */
export function FormActions({
  leading,
  cancelLabel = 'Cancel',
  onCancel,
  cancelDisabled = false,
  confirmLabel,
  confirmVariant = 'default',
  onConfirm,
  confirmDisabled = false,
  loading = false,
}: FormActionsProps) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      {leading}
      <div className="flex gap-2 sm:ml-auto">
        <Button
          variant="outline"
          className="min-h-tap flex-1"
          onClick={onCancel}
          disabled={cancelDisabled}
        >
          {cancelLabel}
        </Button>
        <LoadingButton
          variant={confirmVariant}
          className="min-h-tap flex-1"
          loading={loading}
          disabled={confirmDisabled || loading}
          onClick={onConfirm}
        >
          {confirmLabel}
        </LoadingButton>
      </div>
    </div>
  )
}
