import { useEffect } from 'react'

import { Loader2, XCircle } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'

import type { PinChangeStep } from './use-change-pin'
import { useChangePin } from './use-change-pin'

type ChangePinDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function StepIndicator({ currentStep }: { currentStep: PinChangeStep }) {
  const steps: PinChangeStep[] = ['current', 'new', 'confirm']
  return (
    <div className="mt-6 flex justify-center gap-2">
      {steps.map((s) => {
        let dotColor = 'bg-muted'
        if (s === currentStep) {
          dotColor = 'bg-primary'
        } else if (steps.indexOf(s) < steps.indexOf(currentStep)) {
          dotColor = 'bg-green-500'
        }
        return <div key={s} className={`size-2 rounded-full transition-colors ${dotColor}`} />
      })}
    </div>
  )
}

const STEP_TITLES: Record<PinChangeStep, { title: string; description: string }> = {
  current: {
    title: 'Enter Current PIN',
    description: 'Enter your current 4-digit PIN to continue',
  },
  new: {
    title: 'Enter New PIN',
    description: 'Choose a new 4-digit PIN',
  },
  confirm: {
    title: 'Confirm New PIN',
    description: 'Enter your new PIN again to confirm',
  },
}

function useAutoCompletePin({
  currentPin,
  newPin,
  confirmPin,
  step,
  handlers,
}: {
  currentPin: string
  newPin: string
  confirmPin: string
  step: PinChangeStep
  handlers: {
    handleCurrentPinComplete: (pin: string) => Promise<void>
    handleNewPinComplete: (pin: string) => void
    handleConfirmPinComplete: (pin: string) => Promise<void>
  }
}) {
  useEffect(() => {
    if (currentPin.length === 4 && step === 'current') {
      void handlers.handleCurrentPinComplete(currentPin)
    }
  }, [currentPin, step, handlers])

  useEffect(() => {
    if (newPin.length === 4 && step === 'new') {
      handlers.handleNewPinComplete(newPin)
    }
  }, [newPin, step, handlers])

  useEffect(() => {
    if (confirmPin.length === 4 && step === 'confirm') {
      void handlers.handleConfirmPinComplete(confirmPin)
    }
  }, [confirmPin, step, handlers])
}

export function ChangePinDialog({ open, onOpenChange }: ChangePinDialogProps) {
  const {
    step,
    currentPin,
    setCurrentPin,
    newPin,
    setNewPin,
    confirmPin,
    setConfirmPin,
    error,
    isProcessing,
    resetWizard,
    handleCurrentPinComplete,
    handleNewPinComplete,
    handleConfirmPinComplete,
  } = useChangePin(() => onOpenChange(false))

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen)
    if (!newOpen) {
      resetWizard()
    }
  }

  useAutoCompletePin({
    currentPin,
    newPin,
    confirmPin,
    step,
    handlers: { handleCurrentPinComplete, handleNewPinComplete, handleConfirmPinComplete },
  })

  const pinValue = { current: currentPin, new: newPin, confirm: confirmPin }[step]
  const pinSetter = { current: setCurrentPin, new: setNewPin, confirm: setConfirmPin }[step]

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{STEP_TITLES[step].title}</DialogTitle>
          <DialogDescription>{STEP_TITLES[step].description}</DialogDescription>
        </DialogHeader>

        <PinInputSection
          isProcessing={isProcessing}
          value={pinValue}
          onChange={pinSetter}
          error={error}
          step={step}
        />
      </DialogContent>
    </Dialog>
  )
}

function PinInputSection({
  isProcessing,
  value,
  onChange,
  error,
  step,
}: {
  isProcessing: boolean
  value: string
  onChange: (value: string) => void
  error: string | null
  step: PinChangeStep
}) {
  return (
    <div className="py-4">
      <div className="flex justify-center">
        {isProcessing ? (
          <Loader2 className="text-muted-foreground size-12 animate-spin" />
        ) : (
          <InputOTP maxLength={4} value={value} onChange={onChange}>
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

      <StepIndicator currentStep={step} />
    </div>
  )
}
