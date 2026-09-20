import { useState } from 'react'

import { Group, Row } from '@/components/grouped-list'
import { ChangePinDialog, PasskeyManager } from '@/components/portal'
import { usePasskeySupport } from '@/hooks/portal'

import { WebAuthnRPConfig } from './webauthn-rp-config'

export function AuthenticationSection() {
  const [pinDialogOpen, setPinDialogOpen] = useState(false)
  const { isSupported: passkeySupported } = usePasskeySupport()

  return (
    <>
      <Group header="PIN" footer="The PIN you sign in to the admin app with.">
        <Row title="Change PIN" chevron onClick={() => setPinDialogOpen(true)} />
      </Group>

      <WebAuthnRPConfig />

      {passkeySupported ? (
        <div className="px-screen">
          <PasskeyManager />
        </div>
      ) : null}

      <ChangePinDialog open={pinDialogOpen} onOpenChange={setPinDialogOpen} />
    </>
  )
}
