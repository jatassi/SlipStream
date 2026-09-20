import { Link } from '@tanstack/react-router'
import { Download } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import {
  getActiveClassName,
  getBaseClassName,
  getFlashClassName,
  getGlowClassName,
  getHoverClassName,
  getIconClassName,
} from './downloads-nav-classes'
import type { DownloadsNavLinkProps } from './downloads-nav-types'
import { DownloadsProgressOverlay } from './downloads-progress-overlay'
import { useDownloadsNav } from './use-downloads-nav'

function CountBadge({ movieCount, tvCount }: { movieCount: number; tvCount: number }) {
  return (
    <span className="flex items-center text-xs">
      {movieCount > 0 && <span className="text-movie-400 font-medium">{movieCount}</span>}
      {movieCount > 0 && tvCount > 0 && <span className="text-muted-foreground px-1">|</span>}
      {tvCount > 0 && <span className="text-tv-400 font-medium">{tvCount}</span>}
    </span>
  )
}

function DownloadsLink({
  collapsed,
  indented,
  popover,
}: DownloadsNavLinkProps) {
  const nav = useDownloadsNav()
  const themeFlags = { theme: nav.theme, hasDownloads: nav.hasDownloads }

  return (
    <Link
      to="/downloads"
      aria-current={nav.isActive ? 'page' : undefined}
      aria-label={collapsed ? 'Downloads' : undefined}
      className={cn(
        getBaseClassName({ collapsed, indented: indented ?? false, popover: popover ?? false }),
        getHoverClassName(themeFlags),
        getActiveClassName(nav.isActive, themeFlags),
        getGlowClassName({ ...themeFlags, allPaused: nav.allPaused }),
        getFlashClassName(nav.completionFlash),
      )}
    >
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {nav.hasDownloads && <DownloadsProgressOverlay theme={nav.theme} progress={nav.progress} allPaused={nav.allPaused} />}
      <Download className={getIconClassName(themeFlags)} />
      {collapsed ? null : (
        <>
          <span className="relative z-10 flex-1">Downloads</span>
          {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
          {nav.hasDownloads && <CountBadge movieCount={nav.movieCount} tvCount={nav.tvCount} />}
        </>
      )}
    </Link>
  )
}

export function DownloadsNavLink({
  collapsed,
  indented = false,
  popover = false,
}: DownloadsNavLinkProps) {
  const nav = useDownloadsNav()
  const linkElement = <DownloadsLink collapsed={collapsed} indented={indented} popover={popover} />

  if (collapsed && !popover) {
    return (
      <Tooltip>
        <TooltipTrigger render={linkElement} />
        <TooltipContent side="right">
          <div className="flex items-center gap-2">
            Downloads
            {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
            {nav.hasDownloads && <CountBadge movieCount={nav.movieCount} tvCount={nav.tvCount} />}
            {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
            {nav.hasDownloads && <span className="text-muted-foreground text-xs">({nav.progress.toFixed(0)}%)</span>}
          </div>
        </TooltipContent>
      </Tooltip>
    )
  }

  return linkElement
}
