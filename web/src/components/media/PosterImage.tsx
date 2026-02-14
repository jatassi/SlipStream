import { useState } from 'react'

import { Film, Tv } from 'lucide-react'

import { getLocalArtworkUrl, POSTER_SIZES } from '@/lib/constants'
import { cn } from '@/lib/utils'
import { useArtworkStore } from '@/stores/artwork'

type PosterImageProps = {
  // For TMDB paths - e.g., "/abc123.jpg" (will be prefixed with TMDB base URL)
  path?: string | null
  // For full URLs from search results - e.g., "https://image.tmdb.org/t/p/w500/abc123.jpg"
  url?: string | null
  // For local artwork (library items) - the TMDB ID (primary)
  tmdbId?: number | null
  // For local artwork (library items) - the TVDB ID (fallback when tmdbId is 0)
  tvdbId?: number | null
  alt: string
  size?: keyof typeof POSTER_SIZES
  type?: 'movie' | 'series'
  // Cache-busting version (e.g., updatedAt timestamp) for initial page loads
  version?: string | null
  className?: string
}

export function PosterImage({
  path,
  url,
  tmdbId,
  tvdbId,
  alt,
  size = 'w342',
  type = 'movie',
  version,
  className,
}: PosterImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  // Use TMDB ID if available, otherwise fall back to TVDB ID for artwork lookup
  const artworkId = tmdbId && tmdbId > 0 ? tmdbId : tvdbId && tvdbId > 0 ? tvdbId : null

  // Subscribe to artwork version changes for this specific artwork
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'poster') : 0,
  )

  // Determine if this is a local artwork request
  const isLocalArtwork = !!(artworkId && artworkId > 0)

  // Priority: local artwork (tmdbId or tvdbId) > full URL > TMDB path
  let imageUrl: string | null = null
  if (isLocalArtwork) {
    const baseUrl = getLocalArtworkUrl(type, artworkId, 'poster')
    if (artworkVersion > 0) {
      imageUrl = `${baseUrl}?v=${artworkVersion}`
    } else if (version) {
      imageUrl = `${baseUrl}?v=${version}`
    } else {
      imageUrl = baseUrl
    }
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
        className={cn('bg-muted text-muted-foreground flex items-center justify-center', className)}
      >
        {type === 'movie' ? <Film className="size-12" /> : <Tv className="size-12" />}
      </div>
    )
  }

  return (
    <div className={cn('relative overflow-hidden', className)}>
      {loading ? <div className="bg-muted absolute inset-0 animate-pulse" /> : null}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setError(true)}
        className={cn(
          'size-full object-cover transition-opacity',
          loading ? 'opacity-0' : 'opacity-100',
        )}
      />
    </div>
  )
}
