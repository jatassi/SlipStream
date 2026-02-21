import { Loader2, Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'

import { PasskeyList } from './passkey-list'
import { PasskeyRegistrationForm } from './passkey-registration-form'
import { usePasskeyManager } from './use-passkey-manager'

function LoadingView() {
  return (
    <div className="flex items-center justify-center py-8">
      <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
    </div>
  )
}

function PasskeyHeader({
  isRegistering,
  onAddClick,
}: {
  isRegistering: boolean
  onAddClick: () => void
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h3 className="text-lg font-medium">Passkeys</h3>
        <p className="text-muted-foreground text-sm">
          Use passkeys for faster, more secure sign-in
        </p>
      </div>
      {!isRegistering && (
        <Button variant="outline" size="sm" onClick={onAddClick}>
          <Plus className="mr-2 h-4 w-4" />
          Add Passkey
        </Button>
      )}
    </div>
  )
}

export function PasskeyManager() {
  const hook = usePasskeyManager()

  if (hook.isSupportLoading) {
    return <LoadingView />
  }
  if (!hook.isSupported) {
    return null
  }

  return (
    <div className="space-y-4">
      <PasskeyHeader
        isRegistering={hook.isRegistering}
        onAddClick={() => hook.setIsRegistering(true)}
      />

      {hook.isRegistering ? (
        <PasskeyRegistrationForm
          nameInputRef={hook.nameInputRef}
          newPasskeyName={hook.newPasskeyName}
          onNameChange={hook.setNewPasskeyName}
          pin={hook.pin}
          onPinChange={hook.handlePinChange}
          registerPending={hook.registerPending}
          onCancel={hook.handleCancelRegistration}
        />
      ) : null}

      <PasskeyList
        isLoading={hook.isLoading}
        credentials={hook.credentials}
        editingId={hook.editingId}
        editingName={hook.editingName}
        editInputRef={hook.editInputRef}
        onEditChange={hook.setEditingName}
        onStartEdit={hook.handleStartEdit}
        onSaveEdit={hook.handleSaveEdit}
        onCancelEdit={hook.handleCancelEdit}
        onDelete={hook.handleDelete}
        updatePending={hook.updatePending}
        deletePending={hook.deletePending}
      />
    </div>
  )
}
