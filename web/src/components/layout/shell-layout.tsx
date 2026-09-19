import type { ReactNode } from 'react'

import { Toaster } from '@/components/ui/sonner'
import { useViewport } from '@/hooks/use-viewport'

import { PhoneShell } from './phone-shell'
import { WideShell } from './wide-shell'

const PHONE_TOAST_BOTTOM = 'calc(var(--safe-bottom) + var(--spacing-tab-bar, 56px) + 12px)'

function AppToaster() {
  const shell = useViewport()
  if (shell === 'phone') {
    return <Toaster position="bottom-center" style={{ bottom: PHONE_TOAST_BOTTOM }} />
  }
  return <Toaster position="bottom-right" />
}

export function ShellLayout({ children }: { children: ReactNode }) {
  const shell = useViewport()

  return (
    <>
      <div className="bg-background h-dvh overflow-hidden" data-shell={shell}>
        {shell === 'phone' ? <PhoneShell>{children}</PhoneShell> : <WideShell>{children}</WideShell>}
      </div>
      <AppToaster />
    </>
  )
}
