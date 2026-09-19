/* eslint-disable react-hooks/refs */
import type { AnimationEvent, ReactNode } from 'react'
import { useRef, useState } from 'react'

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
  const layerRef = useRef<HTMLDivElement>(null)
  const [ghost, setGhost] = useState<string | null>(null)
  const html = captureGhost({ exiting, ghost, layer: layerRef.current, setGhost })

  return (
    <div
      ref={layerRef}
      className={cn('phone-layer', !fill && 'scroll p-6')}
      style={fill ? undefined : TAB_STACK_INSET}
      data-exiting={exiting ? '' : undefined}
      data-instant={instant ? '' : undefined}
      onAnimationEnd={(event) => {
        handleLayerAnimationEnd(event, exiting, onExitEnd)
      }}
    >
      {html === null ? (
        <LiveOverlay back={back}>{children}</LiveOverlay>
      ) : (
        <div dangerouslySetInnerHTML={{ __html: html }} />
      )}
    </div>
  )
}

function captureGhost({
  exiting,
  ghost,
  layer,
  setGhost,
}: {
  exiting: boolean
  ghost: string | null
  layer: HTMLDivElement | null
  setGhost: (html: string | null) => void
}): string | null {
  if (!exiting) {
    if (ghost !== null) {
      setGhost(null)
    }
    return null
  }
  if (ghost !== null) {
    return ghost
  }
  if (layer === null) {
    return null
  }
  const html = layer.innerHTML
  setGhost(html)
  return html
}

function LiveOverlay({
  back,
  children,
}: {
  back?: { label: string; onClick: () => void }
  children: ReactNode
}) {
  return (
    <>
      {back === undefined ? null : <BackControl label={back.label} onClick={back.onClick} className="-ml-3" />}
      <ErrorBoundary>{children}</ErrorBoundary>
    </>
  )
}

function handleLayerAnimationEnd(event: AnimationEvent<HTMLDivElement>, exiting: boolean, onExitEnd: () => void) {
  if (exiting && event.target === event.currentTarget) {
    onExitEnd()
  }
}
