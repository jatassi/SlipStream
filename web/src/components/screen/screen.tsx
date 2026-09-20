import type { ReactNode, RefObject } from 'react'

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
  bottomBar?: ReactNode
  scrollRef?: RefObject<HTMLDivElement | null>
}

const COLLAPSE_AT = 40
const PHONE_BOTTOM = 'calc(var(--safe-bottom) + var(--spacing-tab-bar) + 24px)'
const BOTTOM_BAR_SPACE = '68px'

function defaultInset(shell: ViewportShell): string {
  if (shell === 'phone') {
    return PHONE_BOTTOM
  }
  return '24px'
}

type ChromeOptions = {
  largeTitle: boolean
  transparentUntil: number | undefined
  bottomInset: string | undefined
  bottomBar: boolean
}

function useScreenChrome({ largeTitle, transparentUntil, bottomInset, bottomBar }: ChromeOptions) {
  const shell = useViewport()
  const { collapsed, onScroll } = useScreenCollapse(transparentUntil ?? COLLAPSE_AT)
  const base = bottomInset ?? defaultInset(shell)
  return {
    shell,
    collapsed,
    onScroll,
    inset: bottomBar ? `calc(${base} + ${BOTTOM_BAR_SPACE})` : base,
    trailingInBar: shell === 'phone' || collapsed || !largeTitle,
    trailingBesideTitle: shell === 'wide' && largeTitle && !collapsed,
    paddingTop: largeTitle ? 'calc(var(--safe-top) + 44px)' : 0,
    alwaysTitle: !largeTitle && transparentUntil === undefined,
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
  bottomBar,
  scrollRef,
}: ScreenProps) {
  const chrome = useScreenChrome({
    largeTitle,
    transparentUntil,
    bottomInset,
    bottomBar: bottomBar !== undefined,
  })

  return (
    <div className="relative h-full">
      <ScreenBar
        title={title}
        collapsed={chrome.collapsed}
        alwaysTitle={chrome.alwaysTitle}
        back={back}
        trailing={chrome.trailingInBar ? trailing : undefined}
      />
      <div
        ref={scrollRef}
        className="scroll h-full"
        role="region"
        aria-label={title}
        onScroll={chrome.onScroll}
        style={{ paddingTop: chrome.paddingTop, paddingBottom: chrome.inset }}
      >
        <ScreenBody
          largeTitle={largeTitle}
          title={title}
          trailing={chrome.trailingBesideTitle ? trailing : undefined}
        >
          {children}
        </ScreenBody>
      </div>
      {bottomBar === undefined ? null : <ScreenBottomBar shell={chrome.shell}>{bottomBar}</ScreenBottomBar>}
    </div>
  )
}

const PHONE_BAR_CLEARANCE = 'calc(var(--safe-bottom) + var(--spacing-tab-bar))'

function ScreenBottomBar({ shell, children }: { shell: ViewportShell; children: ReactNode }) {
  return (
    <div
      className="material absolute inset-x-0 bottom-0 z-20 shadow-[0_-1px_0_var(--material-edge)]"
      style={{ paddingBottom: shell === 'phone' ? PHONE_BAR_CLEARANCE : undefined }}
    >
      <div className="mx-auto w-full max-w-5xl">{children}</div>
    </div>
  )
}

function ScreenBody({
  largeTitle,
  title,
  trailing,
  children,
}: {
  largeTitle: boolean
  title: string
  trailing?: ReactNode
  children: ReactNode
}) {
  if (!largeTitle) {
    return children
  }
  return (
    <div className="mx-auto w-full max-w-5xl min-h-[calc(100%+16rem)]">
      <LargeTitle title={title} trailing={trailing} />
      {children}
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
  alwaysTitle,
  back,
  trailing,
}: {
  title: string
  collapsed: boolean
  alwaysTitle: boolean
  back?: ScreenProps['back']
  trailing?: ReactNode
}) {
  return (
    <header
      className="screen-bar absolute inset-x-0 top-0 z-20"
      aria-label={title}
      data-collapsed={collapsed ? '' : undefined}
      data-always-title={alwaysTitle ? '' : undefined}
    >
      <div className="screen-bar-bg material" aria-hidden="true" />
      <div className="relative safe-top">
        <div className="relative flex h-11 items-center justify-center px-4">
          {back === undefined ? null : <BackButton label={back.label} onClick={back.onClick} />}
          <span
            className="screen-compact-title text-title font-semibold"
            aria-hidden={alwaysTitle || collapsed ? undefined : true}
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

export function BackControl({
  label,
  onClick,
  className,
}: {
  label: string
  onClick: () => void
  className?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn('press-dim flex h-11 items-center pr-3 pl-1 text-title text-foreground', className)}
    >
      <ChevronLeft className="size-7 -ml-1" strokeWidth={2.25} />
      <span className="-ml-0.5">{label}</span>
    </button>
  )
}

function BackButton({ label, onClick }: { label: string; onClick: () => void }) {
  return <BackControl label={label} onClick={onClick} className="absolute left-1" />
}
