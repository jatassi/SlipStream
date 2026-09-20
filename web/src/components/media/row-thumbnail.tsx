import { PosterImage } from '@/components/media/poster-image'

type RowThumbnailProps = {
  path?: string | null
  url?: string | null
  tmdbId?: number | null
  tvdbId?: number | null
  alt: string
  type?: 'movie' | 'series'
  version?: string | null
  size: 'sm' | 'md'
}

const SIZE_CLASSES: Record<RowThumbnailProps['size'], string> = {
  sm: 'h-12 w-8 rounded-[4px]',
  md: 'rounded-thumb h-15 w-10 shrink-0',
}

/**
 * Row-leading thumbnail: a `PosterImage` sized for a `Row`'s `leading` slot.
 */
export function RowThumbnail({ size, ...poster }: RowThumbnailProps) {
  return <PosterImage {...poster} size="w92" className={SIZE_CLASSES[size]} />
}
