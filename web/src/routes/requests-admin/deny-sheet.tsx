import { useState } from 'react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { ControlStack, TextareaRow } from '@/components/settings/control-row'

type DenySheetProps = {
  title: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onDeny: (reason: string) => void
}

export function DenySheet({ title, open, onOpenChange, onDeny }: DenySheetProps) {
  const [reason, setReason] = useState('')

  const close = () => {
    setReason('')
    onOpenChange(false)
  }

  const handleOpenChange = (next: boolean) => {
    if (next) {
      onOpenChange(true)
      return
    }
    close()
  }

  return (
    <SheetPresenter
      open={open}
      onOpenChange={handleOpenChange}
      title={`Deny ${title}?`}
      description="The requester sees this request as denied. It stays in the Denied list."
      footer={
        <FormActions
          confirmLabel="Deny"
          confirmVariant="destructive"
          onConfirm={() => {
            onDeny(reason.trim())
            close()
          }}
          onCancel={close}
        />
      }
    >
      <ControlStack footer="The reason is optional and is shown to the requester in the portal.">
        <TextareaRow
          label="Reason"
          placeholder="Why this request was denied"
          rows={3}
          value={reason}
          onChange={(event) => {
            setReason(event.target.value)
          }}
        />
      </ControlStack>
    </SheetPresenter>
  )
}
