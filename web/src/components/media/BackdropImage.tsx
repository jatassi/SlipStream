import { useState } from 'react'
import { cn } from '@/lib/utils'
import { BACKDROP_SIZES, getLocalArtworkUrl } from '@/lib/constants'
import { useArtworkStore } from '@/stores/artwork'

interface BackdropImageProps {
  // For TMDB paths (search results) - e.g., "/abc123.jpg"
  path?: string | null
  // For local artwork (library items) - the TMDB ID (primary)
  tmdbId?: number | null
  // For local artwork (library items) - the TVDB ID (fallback when tmdbId is 0)
  tvdbId?: number | null
  // Media type for local artwork lookup
  type?: 'movie' | 'series'
  alt: string
  size?: keyof typeof BACKDROP_SIZES
  className?: string
  overlay?: boolean
}

export function BackdropImage({
  path,
  tmdbId,
  tvdbId,
  type = 'movie',
  alt,
  size = 'w1280',
  className,
  overlay = true,
}: BackdropImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  // Use TMDB ID if available, otherwise fall back to TVDB ID for artwork lookup
  const artworkId = tmdbId && tmdbId > 0 ? tmdbId : tvdbId && tvdbId > 0 ? tvdbId : null

  // Subscribe to artwork version changes for this specific artwork
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'backdrop') : 0
  )

  // Determine if this is a local artwork request
  const isLocalArtwork = !!(artworkId && artworkId > 0)

  // Prefer local artwork if tmdbId/tvdbId is provided, otherwise use TMDB path
  let imageUrl: string | null = null
  if (isLocalArtwork) {
    // Add cache-busting param when artwork version changes
    const baseUrl = getLocalArtworkUrl(type, artworkId!, 'backdrop')
    imageUrl = artworkVersion > 0 ? `${baseUrl}?v=${artworkVersion}` : baseUrl
  } else if (path) {
    imageUrl = `${BACKDROP_SIZES[size]}${path}`
  }

  // Reset state when imageUrl changes (React-recommended pattern)
  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
  }

  if (!imageUrl || error) {
    return (
      <div className={cn('bg-gradient-to-b from-muted to-background', className)} />
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
      {overlay && (
        <div className="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent" />
      )}
    </div>
  )
}
