import { useState, useRef, type ReactNode, type SyntheticEvent } from 'react'
import { getLocalArtworkUrl } from '@/lib/constants'
import { useArtworkStore } from '@/stores/artwork'

interface StudioLogoProps {
  tmdbId?: number | null
  type?: 'movie' | 'series'
  alt: string
  version?: string | null
  fallback?: ReactNode
  className?: string
}

const MAX_HEIGHT = 40
const MAX_WIDTH = 120

export function StudioLogo({
  tmdbId,
  type = 'movie',
  alt,
  version,
  fallback,
  className,
}: StudioLogoProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)
  const [dimensions, setDimensions] = useState<{ width: number; height: number } | null>(null)
  const imgRef = useRef<HTMLImageElement>(null)

  const artworkVersion = useArtworkStore((state) =>
    tmdbId ? state.getVersion(type, tmdbId, 'studio_logo') : 0
  )

  let imageUrl: string | null = null
  if (tmdbId && tmdbId > 0) {
    const baseUrl = getLocalArtworkUrl(type, tmdbId, 'studio_logo')
    if (artworkVersion > 0) {
      imageUrl = `${baseUrl}?v=${artworkVersion}`
    } else if (version) {
      imageUrl = `${baseUrl}?v=${version}`
    } else {
      imageUrl = baseUrl
    }
  }

  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
    setDimensions(null)
  }

  if (!imageUrl || error) {
    return fallback ? <div className={className}>{fallback}</div> : null
  }

  const handleLoad = (e: SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget
    const aspect = img.naturalWidth / img.naturalHeight

    let w: number, h: number
    if (aspect > MAX_WIDTH / MAX_HEIGHT) {
      w = MAX_WIDTH
      h = MAX_WIDTH / aspect
    } else {
      h = MAX_HEIGHT
      w = MAX_HEIGHT * aspect
    }

    setDimensions({ width: w, height: h })
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
        className={`object-contain brightness-0 invert opacity-70 drop-shadow-lg ${loading ? 'hidden' : ''}`}
      />
    </div>
  )
}
