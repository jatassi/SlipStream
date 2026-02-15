import { formatDistanceToNow } from 'date-fns'
import { Check, KeyRound, Pencil, Trash2, X } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

type PasskeyCredential = {
  id: string
  name: string
  createdAt: string
  lastUsedAt: string | null
}

type PasskeyCredentialRowProps = {
  cred: PasskeyCredential
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

function EditingContent({
  editInputRef,
  editingName,
  onEditChange,
  onSaveEdit,
  onCancelEdit,
  updatePending,
}: {
  editInputRef: React.RefObject<HTMLInputElement | null>
  editingName: string
  onEditChange: (value: string) => void
  onSaveEdit: () => void
  onCancelEdit: () => void
  updatePending: boolean
}) {
  return (
    <div className="flex items-center gap-2">
      <Input
        ref={editInputRef}
        value={editingName}
        onChange={(e) => onEditChange(e.target.value)}
        className="h-8 w-48"
      />
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8"
        onClick={onSaveEdit}
        disabled={updatePending}
      >
        <Check className="h-4 w-4" />
      </Button>
      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onCancelEdit}>
        <X className="h-4 w-4" />
      </Button>
    </div>
  )
}

function ViewingContent({ cred }: { cred: PasskeyCredential }) {
  return (
    <div>
      <div className="font-medium">{cred.name}</div>
      <div className="text-muted-foreground text-sm">
        Created {formatDistanceToNow(new Date(cred.createdAt))} ago
        {cred.lastUsedAt ? (
          <> · Last used {formatDistanceToNow(new Date(cred.lastUsedAt))} ago</>
        ) : null}
      </div>
    </div>
  )
}

function ActionButtons({
  credId,
  credName,
  onStartEdit,
  onDelete,
  deletePending,
}: {
  credId: string
  credName: string
  onStartEdit: (id: string, name: string) => void
  onDelete: (id: string) => void
  deletePending: boolean
}) {
  return (
    <div className="flex items-center gap-1">
      <Button variant="ghost" size="icon" onClick={() => onStartEdit(credId, credName)}>
        <Pencil className="h-4 w-4" />
      </Button>
      <Button variant="ghost" size="icon" onClick={() => onDelete(credId)} disabled={deletePending}>
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  )
}

export function PasskeyCredentialRow({
  cred,
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
}: PasskeyCredentialRowProps) {
  const isEditing = editingId === cred.id

  return (
    <div className="border-border flex items-center justify-between rounded-lg border p-3">
      <div className="flex items-center gap-3">
        <KeyRound className="text-muted-foreground h-5 w-5" />
        {isEditing ? (
          <EditingContent
            editInputRef={editInputRef}
            editingName={editingName}
            onEditChange={onEditChange}
            onSaveEdit={onSaveEdit}
            onCancelEdit={onCancelEdit}
            updatePending={updatePending}
          />
        ) : (
          <ViewingContent cred={cred} />
        )}
      </div>
      {!isEditing && (
        <ActionButtons
          credId={cred.id}
          credName={cred.name}
          onStartEdit={onStartEdit}
          onDelete={onDelete}
          deletePending={deletePending}
        />
      )}
    </div>
  )
}
