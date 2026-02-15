import { useState } from 'react'

import { BACKDROP_SIZES, getLocalArtworkUrl } from '@/lib/constants'
import { cn } from '@/lib/utils'
import { useArtworkStore } from '@/stores/artwork'

type BackdropImageProps = {
  path?: string | null
  tmdbId?: number | null
  tvdbId?: number | null
  type?: 'movie' | 'series'
  alt: string
  size?: keyof typeof BACKDROP_SIZES
  version?: string | null
  className?: string
  overlay?: boolean
}

function selectArtworkId(tmdbId?: number | null, tvdbId?: number | null): number | null {
  if (tmdbId && tmdbId > 0) {return tmdbId}
  if (tvdbId && tvdbId > 0) {return tvdbId}
  return null
}

function buildImageUrl(
  params: {
    artworkId: number | null
    type: 'movie' | 'series'
    artworkVersion: number
    version?: string | null
    path?: string | null
    size: keyof typeof BACKDROP_SIZES
  },
): string | null {
  if (params.artworkId) {
    const baseUrl = getLocalArtworkUrl(params.type, params.artworkId, 'backdrop')
    if (params.artworkVersion > 0) {return `${baseUrl}?v=${params.artworkVersion}`}
    if (params.version) {return `${baseUrl}?v=${params.version}`}
    return baseUrl
  }
  if (params.path) {return `${BACKDROP_SIZES[params.size]}${params.path}`}
  return null
}

export function BackdropImage({ path, tmdbId, tvdbId, type = 'movie', alt, size = 'w1280', version, className, overlay = true }: BackdropImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  const artworkId = selectArtworkId(tmdbId, tvdbId)
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'backdrop') : 0,
  )
  const imageUrl = buildImageUrl({ artworkId, type, artworkVersion, version, path, size })

  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
  }

  if (!imageUrl || error) {return <div className={cn('from-muted to-background bg-gradient-to-b', className)} />}

  return (
    <div className={cn('relative overflow-hidden', className)}>
      {loading ? <div className="bg-muted absolute inset-0 animate-pulse" /> : null}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setError(true)}
        className={cn('size-full object-cover transition-opacity', loading ? 'opacity-0' : 'opacity-100')}
      />
      {overlay ? <div className="from-background via-background/60 absolute inset-0 bg-gradient-to-t to-transparent" /> : null}
    </div>
  )
}
