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

export function DownloadsNavLink({
  collapsed,
  indented = false,
  popover = false,
}: DownloadsNavLinkProps) {
  const nav = useDownloadsNav()
  const themeFlags = { theme: nav.theme, hasDownloads: nav.hasDownloads }

  const linkElement = (
    <Link
      to="/activity"
      className={cn(
        getBaseClassName({ collapsed, indented, popover }),
        getHoverClassName(themeFlags),
        getActiveClassName(nav.isActive, themeFlags),
        getGlowClassName({ ...themeFlags, allPaused: nav.allPaused }),
        getFlashClassName(nav.completionFlash),
      )}
    >
      {nav.hasDownloads ? (
        <DownloadsProgressOverlay theme={nav.theme} progress={nav.progress} allPaused={nav.allPaused} />
      ) : null}

      <Download className={getIconClassName(themeFlags)} />
      {!collapsed && (
        <>
          <span className="relative z-10 flex-1">Downloads</span>
          {nav.hasDownloads ? <CountBadge movieCount={nav.movieCount} tvCount={nav.tvCount} /> : null}
        </>
      )}
    </Link>
  )

  if (collapsed && !popover) {
    return (
      <Tooltip>
        <TooltipTrigger render={linkElement} />
        <TooltipContent side="right">
          <div className="flex items-center gap-2">
            Downloads
            {nav.hasDownloads ? <CountBadge movieCount={nav.movieCount} tvCount={nav.tvCount} /> : null}
            {nav.hasDownloads ? (
              <span className="text-muted-foreground text-xs">({nav.progress.toFixed(0)}%)</span>
            ) : null}
          </div>
        </TooltipContent>
      </Tooltip>
    )
  }

  return linkElement
}
