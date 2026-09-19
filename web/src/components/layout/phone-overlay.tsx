import type { AnimationEvent, ReactNode } from 'react'

import { ErrorBoundary } from '@/components/error-boundary'
import { BackControl } from '@/components/screen/screen'
import { cn } from '@/lib/utils'

type PhoneOverlayProps = {
  children: ReactNode
  fill: boolean
  back?: { label: string; onClick: () => void }
  exiting: boolean
  instant: boolean
  onExitEnd: () => void
}

const TAB_STACK_INSET = { paddingBottom: 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)' }

export function PhoneOverlay({ children, fill, back, exiting, instant, onExitEnd }: PhoneOverlayProps) {
  return (
    <div
      className={cn('phone-layer', !fill && 'scroll p-6')}
      style={fill ? undefined : TAB_STACK_INSET}
      data-exiting={exiting ? '' : undefined}
      data-instant={instant ? '' : undefined}
      onAnimationEnd={(event) => {
        handleLayerAnimationEnd(event, exiting, onExitEnd)
      }}
    >
      {back === undefined ? null : <BackControl label={back.label} onClick={back.onClick} className="-ml-3" />}
      <ErrorBoundary>{children}</ErrorBoundary>
    </div>
  )
}

function handleLayerAnimationEnd(event: AnimationEvent<HTMLDivElement>, exiting: boolean, onExitEnd: () => void) {
  if (exiting && event.target === event.currentTarget) {
    onExitEnd()
  }
}
