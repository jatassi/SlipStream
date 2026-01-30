import { useState, useEffect, useMemo, useRef } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import { Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useDownloadingStore } from '@/stores'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

type DownloadTheme = 'movie' | 'tv' | 'both' | 'none'

interface DownloadsNavLinkProps {
  collapsed: boolean
  indented?: boolean
  popover?: boolean
}

export function DownloadsNavLink({ collapsed, indented = false, popover = false }: DownloadsNavLinkProps) {
  const router = useRouterState()
  const queueItems = useDownloadingStore((state) => state.queueItems)
  const [completionFlash, setCompletionFlash] = useState<DownloadTheme | null>(null)
  const prevItemsRef = useRef<Map<string, string>>(new Map())

  const isActive = router.location.pathname === '/activity' ||
    router.location.pathname.startsWith('/activity/')

  const { progress, theme, movieCount, tvCount } = useMemo(() => {
    const movies = queueItems.filter((item) => item.mediaType === 'movie')
    const tv = queueItems.filter((item) => item.mediaType === 'series')
    const movieCount = movies.length
    const tvCount = tv.length

    let theme: DownloadTheme = 'none'
    if (movieCount > 0 && tvCount > 0) theme = 'both'
    else if (movieCount > 0) theme = 'movie'
    else if (tvCount > 0) theme = 'tv'

    const totalSize = queueItems.reduce((acc, item) => acc + (item.size || 0), 0)
    const downloadedSize = queueItems.reduce((acc, item) => acc + (item.downloadedSize || 0), 0)
    const progress = totalSize > 0 ? (downloadedSize / totalSize) * 100 : 0

    return { progress, theme, movieCount, tvCount }
  }, [queueItems])

  // Stable item IDs for comparison (only changes when items are added/removed)
  const itemIds = useMemo(
    () => queueItems.map((i) => `${i.id}:${i.mediaType}`).sort().join(','),
    [queueItems]
  )

  // Detect completions via effect (only runs when itemIds changes, not on progress updates)
  useEffect(() => {
    const currentItems = new Map(
      queueItems.map((i) => [i.id, i.mediaType])
    )
    const prevItems = prevItemsRef.current

    if (prevItems.size > 0 && currentItems.size < prevItems.size) {
      const removedTypes = new Set<'movie' | 'series'>()
      for (const [id, mediaType] of prevItems) {
        if (!currentItems.has(id)) {
          if (mediaType === 'movie') removedTypes.add('movie')
          if (mediaType === 'series') removedTypes.add('series')
        }
      }

      if (removedTypes.size > 0) {
        let flashTheme: DownloadTheme
        if (removedTypes.has('movie') && removedTypes.has('series')) {
          flashTheme = 'both'
        } else if (removedTypes.has('movie')) {
          flashTheme = 'movie'
        } else {
          flashTheme = 'tv'
        }
        // Schedule state update asynchronously to avoid cascading render warning
        queueMicrotask(() => setCompletionFlash(flashTheme))
      }
    }

    prevItemsRef.current = currentItems
  }, [itemIds]) // eslint-disable-line react-hooks/exhaustive-deps

  // Clear flash after timeout
  useEffect(() => {
    if (completionFlash) {
      const timer = setTimeout(() => setCompletionFlash(null), 800)
      return () => clearTimeout(timer)
    }
  }, [completionFlash])

  const hasDownloads = queueItems.length > 0

  const badge = (
    <span className="flex items-center text-xs">
      {movieCount > 0 && (
        <span className="text-movie-400 font-medium">{movieCount}</span>
      )}
      {movieCount > 0 && tvCount > 0 && (
        <span className="px-1 text-muted-foreground">|</span>
      )}
      {tvCount > 0 && (
        <span className="text-tv-400 font-medium">{tvCount}</span>
      )}
    </span>
  )

  const iconClassName = cn(
    'size-4 shrink-0 transition-all duration-300 relative z-10',
    hasDownloads && 'text-white',
    hasDownloads && theme === 'movie' && 'icon-glow-movie',
    hasDownloads && theme === 'tv' && 'icon-glow-tv',
    !hasDownloads && 'text-muted-foreground'
  )

  const linkContent = (
    <>
      <Download className={iconClassName} />
      {!collapsed && (
        <>
          <span className="flex-1 relative z-10">Downloads</span>
          {hasDownloads && badge}
        </>
      )}
    </>
  )

  const baseClassName = cn(
    'relative flex items-center rounded-md text-sm font-medium transition-all duration-300 border-l-2 border-transparent',
    popover ? 'gap-2 px-2 py-1.5 border-l-0' : 'gap-3 px-3 py-2',
    collapsed && !popover && 'justify-center px-2 border-l-0',
    indented && !collapsed && !popover && 'ml-4 border-l border-border pl-4'
  )

  const glowClassName = cn(
    hasDownloads && 'ring-1 ring-inset',
    hasDownloads && theme === 'movie' && 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]',
    hasDownloads && theme === 'tv' && 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
    hasDownloads && theme === 'both' && 'ring-white/20 animate-[inset-glow-pulse-media_2s_ease-in-out_infinite]'
  )

  const flashAnimationClass = cn(
    completionFlash === 'movie' && 'animate-[download-complete-flash-movie_800ms_ease-out]',
    completionFlash === 'tv' && 'animate-[download-complete-flash-tv_800ms_ease-out]',
    completionFlash === 'both' && 'animate-[download-complete-flash-media_800ms_ease-out]'
  )

  const progressBarGradient = cn(
    theme === 'movie' && 'bg-gradient-to-r from-movie-600/40 via-movie-500/50 to-movie-500/60',
    theme === 'tv' && 'bg-gradient-to-r from-tv-600/40 via-tv-500/50 to-tv-500/60',
    theme === 'both' && 'bg-gradient-to-r from-movie-500/50 to-tv-500/50'
  )

  const hoverClassName = cn(
    !hasDownloads && 'hover:bg-accent hover:text-accent-foreground',
    hasDownloads && theme === 'movie' && 'hover:bg-movie-500/15',
    hasDownloads && theme === 'tv' && 'hover:bg-tv-500/15',
    hasDownloads && theme === 'both' && 'hover:bg-accent/50'
  )

  const activeClassName = cn(
    isActive && !hasDownloads && 'bg-accent text-accent-foreground',
    isActive && hasDownloads && theme === 'movie' && 'ring-movie-500/70',
    isActive && hasDownloads && theme === 'tv' && 'ring-tv-500/70',
    isActive && hasDownloads && theme === 'both' && 'ring-white/40'
  )

  const linkElement = (
    <Link
      to="/activity"
      className={cn(baseClassName, hoverClassName, activeClassName, glowClassName, flashAnimationClass)}
    >
      {hasDownloads && (
        <div className="absolute inset-0 overflow-hidden rounded-md">
          <div className="absolute inset-0 bg-muted/30" />

          <div
            className={cn(
              'absolute inset-y-0 left-0 transition-all duration-500 ease-out',
              progressBarGradient
            )}
            style={{ width: `${Math.max(progress, 2)}%` }}
          >
            <div className="absolute inset-0 overflow-hidden">
              <div
                className={cn(
                  'absolute inset-y-0 w-1/3 animate-[shimmer_2.5s_linear_infinite]',
                  theme === 'movie' && 'bg-gradient-to-r from-transparent via-movie-400/20 to-transparent',
                  theme === 'tv' && 'bg-gradient-to-r from-transparent via-tv-400/20 to-transparent',
                  theme === 'both' && 'bg-gradient-to-r from-transparent via-white/10 to-transparent'
                )}
              />
            </div>
          </div>

          <div
            className={cn(
              'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
              theme === 'movie' && 'bg-movie-400',
              theme === 'tv' && 'bg-tv-400',
              theme === 'both' && 'bg-gradient-to-b from-movie-400 to-tv-400'
            )}
            style={{ left: `calc(${Math.max(progress, 2)}% - 2px)` }}
          />
        </div>
      )}

      {linkContent}
    </Link>
  )

  if (collapsed && !popover) {
    return (
      <Tooltip>
        <TooltipTrigger render={linkElement} />
        <TooltipContent side="right">
          <div className="flex items-center gap-2">
            Downloads
            {hasDownloads && badge}
            {hasDownloads && (
              <span className="text-xs text-muted-foreground">
                ({progress.toFixed(0)}%)
              </span>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    )
  }

  return linkElement
}
