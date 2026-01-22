import { useState } from 'react'
import { Film, Tv } from 'lucide-react'
import { cn } from '@/lib/utils'
import { POSTER_SIZES, getLocalArtworkUrl } from '@/lib/constants'
import { useArtworkStore } from '@/stores/artwork'

interface PosterImageProps {
  // For TMDB paths - e.g., "/abc123.jpg" (will be prefixed with TMDB base URL)
  path?: string | null
  // For full URLs from search results - e.g., "https://image.tmdb.org/t/p/w500/abc123.jpg"
  url?: string | null
  // For local artwork (library items) - the TMDB ID
  tmdbId?: number | null
  alt: string
  size?: keyof typeof POSTER_SIZES
  type?: 'movie' | 'series'
  className?: string
}

export function PosterImage({
  path,
  url,
  tmdbId,
  alt,
  size = 'w342',
  type = 'movie',
  className,
}: PosterImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  // Subscribe to artwork version changes for this specific artwork
  const artworkVersion = useArtworkStore((state) =>
    tmdbId ? state.getVersion(type, tmdbId, 'poster') : 0
  )

  // Determine if this is a local artwork request
  const isLocalArtwork = !!(tmdbId && tmdbId > 0)

  // Priority: local artwork (tmdbId) > full URL > TMDB path
  let imageUrl: string | null = null
  if (isLocalArtwork) {
    // Add cache-busting param when artwork version changes
    const baseUrl = getLocalArtworkUrl(type, tmdbId, 'poster')
    imageUrl = artworkVersion > 0 ? `${baseUrl}?v=${artworkVersion}` : baseUrl
  } else if (url) {
    imageUrl = url
  } else if (path) {
    imageUrl = `${POSTER_SIZES[size]}${path}`
  }

  // Reset state when imageUrl changes (React-recommended pattern)
  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
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
