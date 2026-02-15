import { useDeletePasskey, usePasskeyCredentials, usePasskeySupport } from '@/hooks/portal'

import { usePasskeyEditing } from './use-passkey-editing'
import { usePasskeyRegistration } from './use-passkey-registration'

export type PasskeyCredentialRow = {
  id: string
  name: string
  createdAt: string
  lastUsedAt: string | null
}

export function usePasskeyManager() {
  const { isSupported, isLoading: isSupportLoading } = usePasskeySupport()
  const { data: credentials, isLoading } = usePasskeyCredentials()
  const deletePasskey = useDeletePasskey()

  const registration = usePasskeyRegistration()
  const editing = usePasskeyEditing()

  const handleDelete = (id: string) => {
    deletePasskey.mutate(id)
  }

  return {
    newPasskeyName: registration.newPasskeyName,
    setNewPasskeyName: registration.setNewPasskeyName,
    pin: registration.pin,
    isRegistering: registration.isRegistering,
    setIsRegistering: registration.setIsRegistering,
    editingId: editing.editingId,
    editingName: editing.editingName,
    setEditingName: editing.setEditingName,
    nameInputRef: registration.nameInputRef,
    editInputRef: editing.editInputRef,
    isSupported,
    isSupportLoading,
    credentials,
    isLoading,
    registerPending: registration.registerPending,
    updatePending: editing.updatePending,
    deletePending: deletePasskey.isPending,
    handlePinChange: registration.handlePinChange,
    handleStartEdit: editing.handleStartEdit,
    handleSaveEdit: editing.handleSaveEdit,
    handleCancelEdit: editing.handleCancelEdit,
    handleCancelRegistration: registration.handleCancelRegistration,
    handleDelete,
  }
}
