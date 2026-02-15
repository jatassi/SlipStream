import { Loader2 } from 'lucide-react'

import { PasskeyCredentialRow } from './passkey-credential-row'
import type { PasskeyCredentialRow as PasskeyCredentialRowType } from './use-passkey-manager'

type PasskeyListProps = {
  isLoading: boolean
  credentials: PasskeyCredentialRowType[] | undefined
  editingId: string | null
  editingName: string
  editInputRef: React.RefObject<HTMLInputElement | null>
  onEditChange: (value: string) => void
  onStartEdit: (id: string, name: string) => void
  onSaveEdit: () => void
  onCancelEdit: () => void
  onDelete: (id: string) => void
  updatePending: boolean
  deletePending: boolean
}

export function PasskeyList({
  isLoading,
  credentials,
  editingId,
  editingName,
  editInputRef,
  onEditChange,
  onStartEdit,
  onSaveEdit,
  onCancelEdit,
  onDelete,
  updatePending,
  deletePending,
}: PasskeyListProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
      </div>
    )
  }

  if (!credentials || credentials.length === 0) {
    return (
      <div className="border-border text-muted-foreground rounded-lg border border-dashed p-8 text-center">
        No passkeys registered. Add one for faster, more secure login.
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {credentials.map((cred) => (
        <PasskeyCredentialRow
          key={cred.id}
          cred={cred}
          editingId={editingId}
          editingName={editingName}
          editInputRef={editInputRef}
          onEditChange={onEditChange}
          onStartEdit={onStartEdit}
          onSaveEdit={onSaveEdit}
          onCancelEdit={onCancelEdit}
          onDelete={onDelete}
          updatePending={updatePending}
          deletePending={deletePending}
        />
      ))}
    </div>
  )
}
