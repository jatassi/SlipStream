import { type ReactNode, type SyntheticEvent, useRef, useState } from 'react'

import { getLocalArtworkUrl } from '@/lib/constants'
import { useArtworkStore } from '@/stores/artwork'

type StudioLogoProps = {
  tmdbId?: number | null
  type?: 'movie' | 'series'
  alt: string
  version?: string | null
  fallback?: ReactNode
  className?: string
}

const MAX_HEIGHT = 40
const MAX_WIDTH = 120

function calculateDimensions(naturalWidth: number, naturalHeight: number) {
  const aspect = naturalWidth / naturalHeight
  if (aspect > MAX_WIDTH / MAX_HEIGHT) {
    return { width: MAX_WIDTH, height: MAX_WIDTH / aspect }
  }
  return { width: MAX_HEIGHT * aspect, height: MAX_HEIGHT }
}

function buildImageUrl(
  tmdbId: number | null | undefined,
  type: 'movie' | 'series',
  versions: { artwork: number; prop?: string | null },
): string | null {
  if (!tmdbId || tmdbId <= 0) {return null}
  const baseUrl = getLocalArtworkUrl(type, tmdbId, 'studio_logo')
  if (versions.artwork > 0) {return `${baseUrl}?v=${versions.artwork}`}
  if (versions.prop) {return `${baseUrl}?v=${versions.prop}`}
  return baseUrl
}

export function StudioLogo({ tmdbId, type = 'movie', alt, version, fallback, className }: StudioLogoProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)
  const [dimensions, setDimensions] = useState<{ width: number; height: number } | null>(null)
  const imgRef = useRef<HTMLImageElement>(null)

  const artworkVersion = useArtworkStore((state) =>
    tmdbId ? state.getVersion(type, tmdbId, 'studio_logo') : 0,
  )
  const imageUrl = buildImageUrl(tmdbId, type, { artwork: artworkVersion, prop: version })

  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
    setDimensions(null)
  }

  if (!imageUrl || error) {return fallback ? <div className={className}>{fallback}</div> : null}

  const handleLoad = (e: SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget
    setDimensions(calculateDimensions(img.naturalWidth, img.naturalHeight))
    setLoading(false)
  }

  return (
    <div className={className}>
      <img
        ref={imgRef}
        src={imageUrl}
        alt={alt}
        onLoad={handleLoad}
        onError={() => setError(true)}
        style={dimensions ? { width: dimensions.width, height: dimensions.height } : undefined}
        className={`object-contain opacity-70 brightness-0 drop-shadow-lg invert ${loading ? 'hidden' : ''}`}
      />
    </div>
  )
}
