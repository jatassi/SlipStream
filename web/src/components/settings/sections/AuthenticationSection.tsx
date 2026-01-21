import { useState } from 'react'
import { Lock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { PasskeyManager, ChangePinDialog } from '@/components/portal'

export function AuthenticationSection() {
  const [pinDialogOpen, setPinDialogOpen] = useState(false)

  return (
    <div className="space-y-6">
      <div>
        <Label className="text-base">PIN</Label>
        <p className="text-sm text-muted-foreground mb-3">
          Update your account PIN
        </p>
        <Button onClick={() => setPinDialogOpen(true)}>
          <Lock className="size-4 mr-2" />
          Change PIN...
        </Button>
      </div>

      <div className="border-t pt-6">
        <PasskeyManager />
      </div>

      <ChangePinDialog open={pinDialogOpen} onOpenChange={setPinDialogOpen} />
    </div>
  )
}
