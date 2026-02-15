import { useCallback, useEffect, useRef, useState } from 'react'

import { useRegisterPasskey } from '@/hooks/portal'

export function usePasskeyRegistration() {
  const [newPasskeyName, setNewPasskeyName] = useState('')
  const [pin, setPin] = useState('')
  const [isRegistering, setIsRegistering] = useState(false)
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
