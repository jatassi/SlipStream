import type { ControlSize } from './controls-types'

const GAP_BY_SIZE: Record<ControlSize, string> = {
  lg: 'gap-2',
  sm: 'gap-1.5',
  xs: 'gap-1',
}

export function gapForSize(size: ControlSize): string {
  return GAP_BY_SIZE[size]
}

const FULL_WIDTH_HEIGHT: Record<ControlSize, string> = {
  xs: 'h-6',
  sm: 'h-8',
  lg: 'h-9',
}

const FIXED_DIMENSIONS: Record<ControlSize, string> = {
  xs: 'h-6 w-20',
  sm: 'h-8 w-24',
  lg: 'h-9 min-w-32',
}

export function progressDimensions(size: ControlSize, fullWidth?: boolean): string {
  if (fullWidth) {
    return `w-full ${FULL_WIDTH_HEIGHT[size]}`
  }
  return FIXED_DIMENSIONS[size]
}

export function themeColor(theme: 'movie' | 'tv'): string {
  return theme === 'movie' ? 'text-movie-400' : 'text-tv-400'
}
