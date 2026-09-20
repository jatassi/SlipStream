import type { ReactNode } from 'react'

import { SheetPresenter } from '@/components/presenter'
import { Screen } from '@/components/screen/screen'
import { useViewport } from '@/hooks/use-viewport'

export type AddMediaPresenterProps = {
  title: string
  backLabel: string
  onClose: () => void
  actions: ReactNode
  children: ReactNode
}

export function AddMediaPresenter({
  title,
  backLabel,
  onClose,
  actions,
  children,
}: AddMediaPresenterProps) {
  const shell = useViewport()

  if (shell === 'phone') {
    return (
      <Screen
        title={title}
        back={{ label: backLabel, onClick: onClose }}
        bottomBar={<div className="px-screen py-2">{actions}</div>}
      >
        <div className="px-screen">{children}</div>
      </Screen>
    )
  }

  return (
    <SheetPresenter
      open
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
      title={title}
      footer={actions}
    >
      {children}
    </SheetPresenter>
  )
}
