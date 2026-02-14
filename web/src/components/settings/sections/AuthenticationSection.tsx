import { useState } from 'react'

import { Lock } from 'lucide-react'

import { ChangePinDialog, PasskeyManager } from '@/components/portal'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'

export function AuthenticationSection() {
  const [pinDialogOpen, setPinDialogOpen] = useState(false)

  return (
    <div className="space-y-6">
      <div>
        <Label className="text-base">PIN</Label>
        <p className="text-muted-foreground mb-3 text-sm">Update your account PIN</p>
        <Button onClick={() => setPinDialogOpen(true)}>
          <Lock className="mr-2 size-4" />
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
