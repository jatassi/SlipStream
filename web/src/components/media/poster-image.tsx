import { useState } from 'react'

import { Film, Tv } from 'lucide-react'

import { getLocalArtworkUrl, POSTER_SIZES } from '@/lib/constants'
import { cn } from '@/lib/utils'
import { useArtworkStore } from '@/stores/artwork'

type PosterImageProps = {
  path?: string | null
  url?: string | null
  tmdbId?: number | null
  tvdbId?: number | null
  alt: string
  size?: keyof typeof POSTER_SIZES
  type?: 'movie' | 'series'
  version?: string | null
  className?: string
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
    url?: string | null
    path?: string | null
    size: keyof typeof POSTER_SIZES
  },
): string | null {
  if (params.artworkId) {
    const baseUrl = getLocalArtworkUrl(params.type, params.artworkId, 'poster')
    if (params.artworkVersion > 0) {return `${baseUrl}?v=${params.artworkVersion}`}
    if (params.version) {return `${baseUrl}?v=${params.version}`}
    return baseUrl
  }
  if (params.url) {return params.url}
  if (params.path) {return `${POSTER_SIZES[params.size]}${params.path}`}
  return null
}

function FallbackIcon({ type, className }: { type: 'movie' | 'series'; className?: string }) {
  return (
    <div className={cn('bg-muted text-muted-foreground flex items-center justify-center', className)}>
      {type === 'movie' ? <Film className="size-12" /> : <Tv className="size-12" />}
    </div>
  )
}

export function PosterImage({ path, url, tmdbId, tvdbId, alt, size = 'w342', type = 'movie', version, className }: PosterImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  const artworkId = selectArtworkId(tmdbId, tvdbId)
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'poster') : 0,
  )
  const imageUrl = buildImageUrl({ artworkId, type, artworkVersion, version, url, path, size })

  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
  }

  if (!imageUrl || error) {return <FallbackIcon type={type} className={className} />}

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
    </div>
  )
}
