import { useEffect } from 'react'

import { Loader2, XCircle } from 'lucide-react'

import {
  Dialog,
  DialogBody,
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

function useChangePinDialog(onOpenChange: (open: boolean) => void) {
  const state = useChangePin(() => onOpenChange(false))

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen)
    if (!newOpen) {state.resetWizard()}
  }

  useAutoCompletePin({
    currentPin: state.currentPin,
    newPin: state.newPin,
    confirmPin: state.confirmPin,
    step: state.step,
    handlers: {
      handleCurrentPinComplete: state.handleCurrentPinComplete,
      handleNewPinComplete: state.handleNewPinComplete,
      handleConfirmPinComplete: state.handleConfirmPinComplete,
    },
  })

  const pinValue = { current: state.currentPin, new: state.newPin, confirm: state.confirmPin }[state.step]
  const pinSetter = { current: state.setCurrentPin, new: state.setNewPin, confirm: state.setConfirmPin }[state.step]

  return { step: state.step, pinValue, pinSetter, error: state.error, isProcessing: state.isProcessing, handleOpenChange }
}

export function ChangePinDialog({ open, onOpenChange }: ChangePinDialogProps) {
  const s = useChangePinDialog(onOpenChange)

  return (
    <Dialog open={open} onOpenChange={s.handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{STEP_TITLES[s.step].title}</DialogTitle>
          <DialogDescription>{STEP_TITLES[s.step].description}</DialogDescription>
        </DialogHeader>
        <DialogBody>
          <PinInputSection isProcessing={s.isProcessing} value={s.pinValue} onChange={s.pinSetter} error={s.error} step={s.step} />
        </DialogBody>
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
          <InputOTP mask maxLength={4} value={value} onChange={onChange}>
            <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border md:gap-2.5">
              <InputOTPSlot index={0} className="size-10 text-lg md:size-12 md:text-xl" />
              <InputOTPSlot index={1} className="size-10 text-lg md:size-12 md:text-xl" />
              <InputOTPSlot index={2} className="size-10 text-lg md:size-12 md:text-xl" />
              <InputOTPSlot index={3} className="size-10 text-lg md:size-12 md:text-xl" />
            </InputOTPGroup>
          </InputOTP>
        )}
      </div>

      {error ? <div className="text-destructive mt-4 flex items-center justify-center gap-2 text-sm">
          <XCircle className="size-4" />
          {error}
        </div> : null}

      <StepIndicator currentStep={step} />
    </div>
  )
}
