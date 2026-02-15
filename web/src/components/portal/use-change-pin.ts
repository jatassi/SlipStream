import { useState } from 'react'

import { toast } from 'sonner'

import { useUpdatePortalProfile, useVerifyPin } from '@/hooks'

export type PinChangeStep = 'current' | 'new' | 'confirm'

async function verifyCurrentPin(pin: string, verifyPinMutation: ReturnType<typeof useVerifyPin>) {
  const result = await verifyPinMutation.mutateAsync(pin)
  if (!result.valid) {
    throw new Error('Incorrect PIN')
  }
}

async function updatePinInProfile(
  newPin: string,
  updateProfileMutation: ReturnType<typeof useUpdatePortalProfile>,
) {
  await updateProfileMutation.mutateAsync({ password: newPin })
  toast.success('PIN updated successfully')
}

function usePinFormState() {
  const [step, setStep] = useState<PinChangeStep>('current')
  const [currentPin, setCurrentPin] = useState('')
  const [newPin, setNewPin] = useState('')
  const [confirmPin, setConfirmPin] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isProcessing, setIsProcessing] = useState(false)

  const resetWizard = () => {
    setStep('current'); setCurrentPin(''); setNewPin(''); setConfirmPin(''); setError(null); setIsProcessing(false)
  }

  return { step, setStep, currentPin, setCurrentPin, newPin, setNewPin, confirmPin, setConfirmPin, error, setError, isProcessing, setIsProcessing, resetWizard }
}

export function useChangePin(onSuccess: () => void) {
  const updateProfileMutation = useUpdatePortalProfile()
  const verifyPinMutation = useVerifyPin()
  const state = usePinFormState()

  const handleCurrentPinComplete = async (pin: string) => {
    if (pin.length !== 4 || state.isProcessing) { return }
    state.setIsProcessing(true); state.setError(null)
    try { await verifyCurrentPin(pin, verifyPinMutation); state.setStep('new') }
    catch { state.setError('Failed to verify PIN'); state.setCurrentPin('') }
    finally { state.setIsProcessing(false) }
  }

  const handleNewPinComplete = (pin: string) => {
    if (pin.length !== 4) { return }
    if (pin === state.currentPin) { state.setError('New PIN must be different from current PIN'); state.setNewPin(''); return }
    state.setError(null); state.setStep('confirm')
  }

  const handleConfirmPinComplete = async (pin: string) => {
    if (pin.length !== 4 || state.isProcessing) { return }
    if (pin !== state.newPin) { state.setError('PINs do not match'); state.setConfirmPin(''); return }
    state.setIsProcessing(true); state.setError(null)
    try { await updatePinInProfile(state.newPin, updateProfileMutation); onSuccess(); state.resetWizard() }
    catch { state.setError('Failed to update PIN'); state.setConfirmPin(''); state.setIsProcessing(false) }
  }

  return {
    ...state,
    handleCurrentPinComplete, handleNewPinComplete, handleConfirmPinComplete,
  }
}
