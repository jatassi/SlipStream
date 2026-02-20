import { cn } from '@/lib/utils'

import type { DownloadTheme } from './downloads-nav-types'

type ThemeFlags = {
  theme: DownloadTheme
  hasDownloads: boolean
  allPaused: boolean
}

type LayoutFlags = {
  collapsed: boolean
  indented: boolean
  popover: boolean
}

export function getIconClassName({ theme, hasDownloads }: Omit<ThemeFlags, 'allPaused'>) {
  return cn(
    'size-4 shrink-0 transition-all duration-300 relative z-10',
    hasDownloads && 'text-white',
    hasDownloads && theme === 'movie' && 'icon-glow-movie',
    hasDownloads && theme === 'tv' && 'icon-glow-tv',
    !hasDownloads && 'text-foreground',
  )
}

export function getBaseClassName({ collapsed, indented, popover }: LayoutFlags) {
  return cn(
    'relative flex items-center rounded-md text-sm font-medium transition-all duration-300 border-l-2 border-transparent',
    popover ? 'gap-2 px-2 py-1.5 border-l-0' : 'gap-3 px-3 py-2',
    collapsed && !popover && 'justify-center px-2 border-l-0',
    indented && !collapsed && !popover && 'ml-4 border-l border-border pl-4',
  )
}

export function getGlowClassName({ theme, hasDownloads, allPaused }: ThemeFlags) {
  return cn(
    hasDownloads && 'ring-1 ring-inset',
    hasDownloads && theme === 'movie' && 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]',
    hasDownloads && theme === 'tv' && 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
    hasDownloads && theme === 'both' && 'ring-white/20 animate-[inset-glow-pulse-media_2s_ease-in-out_infinite]',
    allPaused && 'animation-paused',
  )
}

export function getFlashClassName(completionFlash: DownloadTheme | null) {
  return cn(
    completionFlash === 'movie' && 'animate-[download-complete-flash-movie_800ms_ease-out]',
    completionFlash === 'tv' && 'animate-[download-complete-flash-tv_800ms_ease-out]',
    completionFlash === 'both' && 'animate-[download-complete-flash-media_800ms_ease-out]',
  )
}

export function getProgressBarGradient(theme: DownloadTheme) {
  return cn(
    theme === 'movie' && 'bg-gradient-to-r from-movie-600/40 via-movie-500/50 to-movie-500/60',
    theme === 'tv' && 'bg-gradient-to-r from-tv-600/40 via-tv-500/50 to-tv-500/60',
    theme === 'both' && 'bg-gradient-to-r from-movie-500/50 to-tv-500/50',
  )
}

export function getHoverClassName({ theme, hasDownloads }: Omit<ThemeFlags, 'allPaused'>) {
  return cn(
    !hasDownloads && 'hover:bg-accent hover:text-accent-foreground',
    hasDownloads && theme === 'movie' && 'hover:bg-movie-500/15',
    hasDownloads && theme === 'tv' && 'hover:bg-tv-500/15',
    hasDownloads && theme === 'both' && 'hover:bg-accent/50',
  )
}

export function getActiveClassName(isActive: boolean, { theme, hasDownloads }: Omit<ThemeFlags, 'allPaused'>) {
  if (!isActive) {
    return undefined
  }
  if (!hasDownloads) {
    return 'bg-accent text-accent-foreground'
  }
  return cn(
    theme === 'movie' && 'ring-movie-500/70',
    theme === 'tv' && 'ring-tv-500/70',
    theme === 'both' && 'ring-white/40',
  )
}

export function getShimmerClassName(theme: DownloadTheme) {
  return cn(
    'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
    theme === 'movie' && 'via-movie-400/25 bg-gradient-to-r from-transparent to-transparent',
    theme === 'tv' && 'via-tv-400/25 bg-gradient-to-r from-transparent to-transparent',
    theme === 'both' && 'bg-gradient-to-r from-transparent via-white/15 to-transparent',
  )
}

export function getEdgeGlowClassName(theme: DownloadTheme) {
  return cn(
    'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
    theme === 'movie' && 'bg-movie-400',
    theme === 'tv' && 'bg-tv-400',
    theme === 'both' && 'from-movie-400 to-tv-400 bg-gradient-to-b',
  )
}
