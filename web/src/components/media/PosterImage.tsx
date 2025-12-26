import { useState } from 'react'
import { Film, Tv } from 'lucide-react'
import { cn } from '@/lib/utils'
import { POSTER_SIZES, getLocalArtworkUrl } from '@/lib/constants'

interface PosterImageProps {
  // For TMDB paths (search results) - e.g., "/abc123.jpg"
  path?: string | null
  // For local artwork (library items) - the TMDB ID
  tmdbId?: number | null
  alt: string
  size?: keyof typeof POSTER_SIZES
  type?: 'movie' | 'series'
  className?: string
}

export function PosterImage({
  path,
  tmdbId,
  alt,
  size = 'w342',
  type = 'movie',
  className,
}: PosterImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)

  // Prefer local artwork if tmdbId is provided, otherwise use TMDB path
  let imageUrl: string | null = null
  if (tmdbId && tmdbId > 0) {
    imageUrl = getLocalArtworkUrl(type, tmdbId, 'poster')
  } else if (path) {
    imageUrl = `${POSTER_SIZES[size]}${path}`
  }

  if (!imageUrl || error) {
    return (
      <div
        className={cn(
          'flex items-center justify-center bg-muted text-muted-foreground',
          className
        )}
      >
        {type === 'movie' ? (
          <Film className="size-12" />
        ) : (
          <Tv className="size-12" />
        )}
      </div>
    )
  }

  return (
    <div className={cn('relative overflow-hidden', className)}>
      {loading && (
        <div className="absolute inset-0 animate-pulse bg-muted" />
      )}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setError(true)}
        className={cn(
          'size-full object-cover transition-opacity',
          loading ? 'opacity-0' : 'opacity-100'
        )}
      />
    </div>
  )
}
