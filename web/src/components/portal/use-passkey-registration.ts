import { useCallback, useEffect, useRef, useState } from 'react'

import { useRegisterPasskey } from '@/hooks/portal'

const AUTO_OPEN_HASH = '#new-passkey'

function consumeAutoOpenHash(): boolean {
  if (globalThis.location.hash !== AUTO_OPEN_HASH) {
    return false
  }
  globalThis.history.replaceState(null, '', globalThis.location.pathname + globalThis.location.search)
  return true
}

export function usePasskeyRegistration() {
  const [newPasskeyName, setNewPasskeyName] = useState('')
  const [pin, setPin] = useState('')
  const [isRegistering, setIsRegistering] = useState(consumeAutoOpenHash)
  const isSubmittingRef = useRef(false)
  const nameInputRef = useRef<HTMLInputElement>(null)
  const registerPasskey = useRegisterPasskey()

  useEffect(() => {
    if (isRegistering) {nameInputRef.current?.focus()}
  }, [isRegistering])

  const resetForm = useCallback(() => {
    setNewPasskeyName('')
    setPin('')
    setIsRegistering(false)
  }, [])

  const handleRegister = useCallback(
    async (pinValue: string, nameValue: string) => {
      const canSubmit =
        nameValue.trim() && pinValue.length === 4 && !registerPasskey.isPending && !isSubmittingRef.current
      if (!canSubmit) {
        return
      }
      isSubmittingRef.current = true
      try {
        await registerPasskey.mutateAsync({ pin: pinValue, name: nameValue })
        resetForm()
      } catch {
        setPin('')
      } finally {
        isSubmittingRef.current = false
      }
    },
    [registerPasskey, resetForm],
  )

  const handlePinChange = useCallback(
    (value: string) => {
      setPin(value)
      if (value.length === 4 && newPasskeyName.trim()) {void handleRegister(value, newPasskeyName)}
    },
    [handleRegister, newPasskeyName],
  )

  return {
    newPasskeyName, setNewPasskeyName, pin, isRegistering, setIsRegistering, nameInputRef,
    registerPending: registerPasskey.isPending, handlePinChange, handleCancelRegistration: resetForm,
  }
}
