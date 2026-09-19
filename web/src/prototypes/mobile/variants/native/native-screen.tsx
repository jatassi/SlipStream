import type { ReactNode, UIEvent } from 'react'
import { useState } from 'react'

import { ChevronLeft } from 'lucide-react'

import { cn } from '@/lib/utils'

type NativeScreenProps = {
  title: string
  children: ReactNode
  back?: { label: string; onClick: () => void }
  right?: ReactNode
  largeTitle?: boolean
  transparentUntil?: number
  bottomInset?: string
}

const COLLAPSE_AT = 40

export function NativeScreen({
  title,
  children,
  back,
  right,
  largeTitle = true,
  transparentUntil,
  bottomInset = 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)',
}: NativeScreenProps) {
  const [collapsed, setCollapsed] = useState(false)
  const threshold = transparentUntil ?? COLLAPSE_AT

  const onScroll = (e: UIEvent<HTMLDivElement>) => {
    setCollapsed(e.currentTarget.scrollTop > threshold)
  }

  return (
    <div className="relative h-full">
      <header
        className="native-bar absolute inset-x-0 top-0 z-20"
        data-collapsed={collapsed ? '' : undefined}
        data-always-title={largeTitle ? undefined : ''}
      >
        <div className="native-bar-bg m-material" />
        <div className="relative m-safe-top">
          <div className="relative flex h-11 items-center justify-center px-4">
            {back !== undefined && <BackButton label={back.label} onClick={back.onClick} />}
            <span className="native-compact-title text-m-title font-semibold">{title}</span>
            {right !== undefined && <div className="absolute right-2 flex items-center">{right}</div>}
          </div>
        </div>
      </header>
      <div
        className="m-scroll h-full"
        onScroll={onScroll}
        style={{ paddingTop: largeTitle ? 'calc(var(--safe-top) + 44px)' : 0, paddingBottom: bottomInset }}
      >
        {largeTitle ? <h1 className="px-screen pt-1 pb-3 text-m-display font-bold">{title}</h1> : null}
        {children}
      </div>
    </div>
  )
}

function BackButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'm-press-dim absolute left-1 flex h-11 items-center pr-3 pl-1 text-m-title text-foreground',
      )}
    >
      <ChevronLeft className="size-7 -ml-1" strokeWidth={2.25} />
      <span className="-ml-0.5">{label}</span>
    </button>
  )
}
