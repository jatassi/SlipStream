import type { ReactNode } from 'react'

import { Toaster } from '@/components/ui/sonner'
import { useViewport } from '@/hooks/use-viewport'

import { PhoneShell } from './phone-shell'
import { WideShell } from './wide-shell'

const PHONE_TOAST_STYLE =
  '[data-sonner-toaster][data-y-position="bottom"]{bottom:calc(env(safe-area-inset-bottom, 0px) + 56px + 12px)!important}'

function AppToaster() {
  const shell = useViewport()
  if (shell === 'phone') {
    return (
      <>
        <style>{PHONE_TOAST_STYLE}</style>
        <Toaster position="bottom-center" />
      </>
    )
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
