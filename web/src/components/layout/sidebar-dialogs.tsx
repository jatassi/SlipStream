import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

function RestartButtonContent({ countdown, isPending }: { countdown: number | null; isPending: boolean }) {
  if (countdown !== null) {
    return (
      <>
        <Loader2 className="size-4 animate-spin" />
        Restarting ({countdown}s)
      </>
    )
  }
  if (isPending) {
    return (
      <>
        <Loader2 className="size-4 animate-spin" />
        Restarting...
      </>
    )
  }
  return <>Restart</>
}

type RestartDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onRestart: () => void
  countdown: number | null
  isPending: boolean
}

export function RestartDialog({ open, onOpenChange, onRestart, countdown, isPending }: RestartDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (countdown === null) {
          onOpenChange(nextOpen)
        }
      }}
    >
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Confirm Restart</DialogTitle>
          <DialogDescription>
            {countdown === null
              ? 'Are you sure you want to restart the server? The application will be briefly unavailable.'
              : 'Server is restarting. Page will refresh automatically.'}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={countdown !== null}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onRestart}
            disabled={isPending || countdown !== null}
          >
            <RestartButtonContent countdown={countdown} isPending={isPending} />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type LogoutDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onLogout: () => void
}

export function LogoutDialog({ open, onOpenChange, onLogout }: LogoutDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Confirm Logout</DialogTitle>
          <DialogDescription>Are you sure you want to log out?</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button className="bg-amber-500 text-white hover:bg-amber-600" onClick={onLogout}>
            Logout
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
