import { useCallback, useEffect, useState } from 'react'

import { Loader2, XCircle } from 'lucide-react'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { useUpdatePortalProfile, useVerifyPin } from '@/hooks'

type PinChangeStep = 'current' | 'new' | 'confirm'

type ChangePinDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChangePinDialog({ open, onOpenChange }: ChangePinDialogProps) {
  const updateProfileMutation = useUpdatePortalProfile()
  const verifyPinMutation = useVerifyPin()

  const [step, setStep] = useState<PinChangeStep>('current')
  const [currentPin, setCurrentPin] = useState('')
  const [newPin, setNewPin] = useState('')
  const [confirmPin, setConfirmPin] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isProcessing, setIsProcessing] = useState(false)

  const resetWizard = useCallback(() => {
    setStep('current')
    setCurrentPin('')
    setNewPin('')
    setConfirmPin('')
    setError(null)
    setIsProcessing(false)
  }, [])

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen)
    if (!newOpen) {
      resetWizard()
    }
  }

  const handleCurrentPinComplete = useCallback(
    async (pin: string) => {
      if (pin.length !== 4 || isProcessing) {
        return
      }

      setIsProcessing(true)
      setError(null)

      try {
        const result = await verifyPinMutation.mutateAsync(pin)
        if (result.valid) {
          setStep('new')
        } else {
          setError('Incorrect PIN')
          setCurrentPin('')
        }
      } catch {
        setError('Failed to verify PIN')
        setCurrentPin('')
      } finally {
        setIsProcessing(false)
      }
    },
    [verifyPinMutation, isProcessing],
  )

  const handleNewPinComplete = useCallback(
    (pin: string) => {
      if (pin.length !== 4) {
        return
      }

      if (pin === currentPin) {
        setError('New PIN must be different from current PIN')
        setNewPin('')
        return
      }

      setError(null)
      setStep('confirm')
    },
    [currentPin],
  )

  const handleConfirmPinComplete = useCallback(
    async (pin: string) => {
      if (pin.length !== 4 || isProcessing) {
        return
      }

      if (pin !== newPin) {
        setError('PINs do not match')
        setConfirmPin('')
        return
      }

      setIsProcessing(true)
      setError(null)

      try {
        await updateProfileMutation.mutateAsync({ password: newPin })
        toast.success('PIN updated successfully')
        onOpenChange(false)
        resetWizard()
      } catch {
        setError('Failed to update PIN')
        setConfirmPin('')
        setIsProcessing(false)
      }
    },
    [newPin, updateProfileMutation, resetWizard, isProcessing, onOpenChange],
  )

  useEffect(() => {
    if (currentPin.length === 4 && step === 'current') {
      handleCurrentPinComplete(currentPin)
    }
  }, [currentPin, step, handleCurrentPinComplete])

  useEffect(() => {
    if (newPin.length === 4 && step === 'new') {
      handleNewPinComplete(newPin)
    }
  }, [newPin, step, handleNewPinComplete])

  useEffect(() => {
    if (confirmPin.length === 4 && step === 'confirm') {
      handleConfirmPinComplete(confirmPin)
    }
  }, [confirmPin, step, handleConfirmPinComplete])

  const stepConfig = {
    current: {
      title: 'Enter Current PIN',
      description: 'Enter your current 4-digit PIN to continue',
      value: currentPin,
      onChange: setCurrentPin,
    },
    new: {
      title: 'Enter New PIN',
      description: 'Choose a new 4-digit PIN',
      value: newPin,
      onChange: setNewPin,
    },
    confirm: {
      title: 'Confirm New PIN',
      description: 'Enter your new PIN again to confirm',
      value: confirmPin,
      onChange: setConfirmPin,
    },
  }

  const config = stepConfig[step]

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{config.title}</DialogTitle>
          <DialogDescription>{config.description}</DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <div className="flex justify-center">
            {isProcessing ? (
              <Loader2 className="text-muted-foreground size-12 animate-spin" />
            ) : (
              <InputOTP maxLength={4} value={config.value} onChange={config.onChange}>
                <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
                  <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
                  <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
                  <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
                  <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
                </InputOTPGroup>
              </InputOTP>
            )}
          </div>

          {error ? (
            <div className="text-destructive mt-4 flex items-center justify-center gap-2 text-sm">
              <XCircle className="size-4" />
              {error}
            </div>
          ) : null}

          <div className="mt-6 flex justify-center gap-2">
            {(['current', 'new', 'confirm'] as const).map((s) => (
              <div
                key={s}
                className={`size-2 rounded-full transition-colors ${
                  s === step
                    ? 'bg-primary'
                    : ['current', 'new', 'confirm'].indexOf(s) <
                        ['current', 'new', 'confirm'].indexOf(step)
                      ? 'bg-green-500'
                      : 'bg-muted'
                }`}
              />
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
