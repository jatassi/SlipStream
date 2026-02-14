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

  const artworkId = tmdbId && tmdbId > 0 ? tmdbId : tvdbId && tvdbId > 0 ? tvdbId : null

  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'logo') : 0,
  )

  let imageUrl: string | null = null
  if (artworkId && artworkId > 0) {
    const baseUrl = getLocalArtworkUrl(type, artworkId, 'logo')
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
  }

  if (!imageUrl || error) {
    return <>{fallback}</>
  }

  return (
    <div className={className}>
      {loading ? fallback : null}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setError(true)}
        className={
          loading
            ? 'hidden'
            : 'max-h-16 w-auto object-contain object-left drop-shadow-lg md:max-h-20'
        }
      />
    </div>
  )
}
