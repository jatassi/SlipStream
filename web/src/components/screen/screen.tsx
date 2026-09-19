import type { ReactNode } from 'react'

import { ChevronLeft } from 'lucide-react'

import type { ViewportShell } from '@/hooks/use-viewport'
import { useViewport } from '@/hooks/use-viewport'
import { cn } from '@/lib/utils'

import { useScreenCollapse } from './use-screen-collapse'

import './screen.css'

export type ScreenProps = {
  title: string
  children: ReactNode
  back?: { label: string; onClick: () => void }
  trailing?: ReactNode
  largeTitle?: boolean
  transparentUntil?: number
  bottomInset?: string
}

const COLLAPSE_AT = 40
const PHONE_BOTTOM = 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)'

function defaultInset(shell: ViewportShell): string {
  if (shell === 'phone') {
    return PHONE_BOTTOM
  }
  return '24px'
}

function useScreenChrome(largeTitle: boolean, transparentUntil: number | undefined, bottomInset: string | undefined) {
  const shell = useViewport()
  const { collapsed, onScroll } = useScreenCollapse(transparentUntil ?? COLLAPSE_AT)
  return {
    collapsed,
    onScroll,
    inset: bottomInset ?? defaultInset(shell),
    trailingInBar: shell === 'phone' || collapsed || !largeTitle,
    trailingBesideTitle: shell === 'wide' && largeTitle && !collapsed,
    paddingTop: largeTitle ? 'calc(var(--safe-top) + 44px)' : 0,
  }
}

export function Screen({
  title,
  children,
  back,
  trailing,
  largeTitle = true,
  transparentUntil,
  bottomInset,
}: ScreenProps) {
  const chrome = useScreenChrome(largeTitle, transparentUntil, bottomInset)

  return (
    <div className="relative h-full">
      <ScreenBar
        title={title}
        collapsed={chrome.collapsed}
        largeTitle={largeTitle}
        back={back}
        trailing={chrome.trailingInBar ? trailing : undefined}
      />
      <div
        className="scroll h-full"
        onScroll={chrome.onScroll}
        style={{ paddingTop: chrome.paddingTop, paddingBottom: chrome.inset }}
      >
        <div className={cn('mx-auto w-full max-w-5xl', largeTitle && 'min-h-[calc(100%+16rem)]')}>
          {largeTitle ? (
            <LargeTitle title={title} trailing={chrome.trailingBesideTitle ? trailing : undefined} />
          ) : null}
          {children}
        </div>
      </div>
    </div>
  )
}

function LargeTitle({ title, trailing }: { title: string; trailing?: ReactNode }) {
  return (
    <div className="flex items-end justify-between gap-3 px-screen pt-1 pb-3">
      <h1 className="text-display font-bold">{title}</h1>
      {trailing === undefined ? null : <div className="flex items-center pb-1">{trailing}</div>}
    </div>
  )
}

function ScreenBar({
  title,
  collapsed,
  largeTitle,
  back,
  trailing,
}: {
  title: string
  collapsed: boolean
  largeTitle: boolean
  back?: ScreenProps['back']
  trailing?: ReactNode
}) {
  return (
    <header
      className="screen-bar absolute inset-x-0 top-0 z-20"
      aria-label={title}
      data-collapsed={collapsed ? '' : undefined}
      data-always-title={largeTitle ? undefined : ''}
    >
      <div className="screen-bar-bg material" aria-hidden="true" />
      <div className="relative safe-top">
        <div className="relative flex h-11 items-center justify-center px-4">
          {back === undefined ? null : <BackButton label={back.label} onClick={back.onClick} />}
          <span
            className="screen-compact-title text-title font-semibold"
            aria-hidden={largeTitle && !collapsed ? true : undefined}
          >
            {title}
          </span>
          {trailing === undefined ? null : (
            <div className="absolute right-2 flex items-center">{trailing}</div>
          )}
        </div>
      </div>
    </header>
  )
}

function BackButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn('press-dim absolute left-1 flex h-11 items-center pr-3 pl-1 text-title text-foreground')}
    >
      <ChevronLeft className="size-7 -ml-1" strokeWidth={2.25} />
      <span className="-ml-0.5">{label}</span>
    </button>
  )
}
