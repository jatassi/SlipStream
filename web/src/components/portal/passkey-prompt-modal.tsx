import { useState } from 'react'

import { Fingerprint } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'

type PasskeyPromptModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onDismiss: (dontShowAgain: boolean) => void
  onCreate: (dontShowAgain: boolean) => void
}

export function PasskeyPromptModal({ open, onOpenChange, onDismiss, onCreate }: PasskeyPromptModalProps) {
  const [dontShowAgain, setDontShowAgain] = useState(false)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="bg-primary/10 text-primary mx-auto mb-2 flex size-12 items-center justify-center rounded-full">
            <Fingerprint className="size-6" />
          </div>
          <DialogTitle className="text-center text-lg">Set up a passkey?</DialogTitle>
          <DialogDescription className="text-center">
            Skip the PIN — sign in with your face, fingerprint, or device password.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <div className="flex items-center gap-2 px-1 pt-2">
            <Checkbox
              id="passkey-prompt-dont-show"
              checked={dontShowAgain}
              onCheckedChange={setDontShowAgain}
            />
            <Label htmlFor="passkey-prompt-dont-show" className="cursor-pointer font-normal">
              Don&rsquo;t show this again
            </Label>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onDismiss(dontShowAgain)}>
            Not now
          </Button>
          <Button onClick={() => onCreate(dontShowAgain)}>Create</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
