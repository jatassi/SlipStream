import { type ReactNode, useState } from 'react'

import { getLocalArtworkUrl } from '@/lib/constants'
import { useArtworkStore } from '@/stores/artwork'

type TitleTreatmentProps = {
  tmdbId?: number | null
  tvdbId?: number | null
  type?: 'movie' | 'series'
  alt: string
  version?: string | null
  fallback: ReactNode
  className?: string
}

function selectArtworkId(tmdbId?: number | null, tvdbId?: number | null): number | null {
  if (tmdbId && tmdbId > 0) {return tmdbId}
  if (tvdbId && tvdbId > 0) {return tvdbId}
  return null
}

function buildImageUrl(
  artworkId: number | null,
  type: 'movie' | 'series',
  versions: { artwork: number; prop?: string | null },
): string | null {
  if (!artworkId) {return null}
  const baseUrl = getLocalArtworkUrl(type, artworkId, 'logo')
  if (versions.artwork > 0) {return `${baseUrl}?v=${versions.artwork}`}
  if (versions.prop) {return `${baseUrl}?v=${versions.prop}`}
  return baseUrl
}

export function TitleTreatment({
  tmdbId,
  tvdbId,
  type = 'movie',
  alt,
  version,
  fallback,
  className,
}: TitleTreatmentProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [prevImageUrl, setPrevImageUrl] = useState<string | null>(null)

  const artworkId = selectArtworkId(tmdbId, tvdbId)
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'logo') : 0,
  )
  const imageUrl = buildImageUrl(artworkId, type, { artwork: artworkVersion, prop: version })

  if (imageUrl !== prevImageUrl) {
    setPrevImageUrl(imageUrl)
    setError(false)
    setLoading(true)
  }

  if (!imageUrl || error) {return fallback}

  return (
    <div className={className}>
      {loading ? fallback : null}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setError(true)}
        className={loading ? 'hidden' : 'max-h-16 w-auto object-contain object-left drop-shadow-lg md:max-h-20'}
      />
    </div>
  )
}
