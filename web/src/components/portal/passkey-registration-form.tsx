import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { InputOTP, InputOTPGroup, InputOTPSlot } from '@/components/ui/input-otp'
import { Label } from '@/components/ui/label'

type PasskeyRegistrationFormProps = {
  nameInputRef: React.RefObject<HTMLInputElement | null>
  newPasskeyName: string
  onNameChange: (value: string) => void
  pin: string
  onPinChange: (value: string) => void
  registerPending: boolean
  onCancel: () => void
}

export function PasskeyRegistrationForm({
  nameInputRef,
  newPasskeyName,
  onNameChange,
  pin,
  onPinChange,
  registerPending,
  onCancel,
}: PasskeyRegistrationFormProps) {
  return (
    <div className="border-border space-y-4 rounded-lg border p-4">
      <div className="space-y-2">
        <Label>Passkey Name</Label>
        <Input
          ref={nameInputRef}
          placeholder="e.g., MacBook Touch ID"
          value={newPasskeyName}
          onChange={(e) => onNameChange(e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>Enter PIN to confirm</Label>
        <div className="flex justify-center">
          {registerPending ? (
            <Loader2 className="text-muted-foreground h-10 w-10 animate-spin" />
          ) : (
            <InputOTP
              mask maxLength={4}
              value={pin}
              onChange={onPinChange}
              disabled={!newPasskeyName.trim()}
            >
              <InputOTPGroup className="gap-2 *:data-[slot=input-otp-slot]:rounded-md *:data-[slot=input-otp-slot]:border">
                <InputOTPSlot index={0} className="size-10 text-lg" />
                <InputOTPSlot index={1} className="size-10 text-lg" />
                <InputOTPSlot index={2} className="size-10 text-lg" />
                <InputOTPSlot index={3} className="size-10 text-lg" />
              </InputOTPGroup>
            </InputOTP>
          )}
        </div>
      </div>
      <div className="flex justify-end">
        <Button variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  )
}
