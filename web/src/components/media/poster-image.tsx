import { useEffect, useRef, useState } from 'react'

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
  onAllFailed?: () => void
}

function selectArtworkId(tmdbId?: number | null, tvdbId?: number | null): number | null {
  if (tmdbId && tmdbId > 0) {return tmdbId}
  if (tvdbId && tvdbId > 0) {return tvdbId}
  return null
}

function buildImageUrls(
  params: {
    artworkId: number | null
    type: 'movie' | 'series'
    artworkVersion: number
    version?: string | null
    url?: string | null
    path?: string | null
    size: keyof typeof POSTER_SIZES
  },
): string[] {
  const urls: string[] = []

  if (params.artworkId) {
    const baseUrl = getLocalArtworkUrl(params.type, params.artworkId, 'poster')
    if (params.artworkVersion > 0) {
      urls.push(`${baseUrl}?v=${params.artworkVersion}`)
    } else if (params.version) {
      urls.push(`${baseUrl}?v=${params.version}`)
    } else {
      urls.push(baseUrl)
    }
  }

  if (params.url) {urls.push(params.url)}
  if (params.path) {urls.push(`${POSTER_SIZES[params.size]}${params.path}`)}

  return urls
}

function FallbackIcon({ type, className, onMount }: { type: 'movie' | 'series'; className?: string; onMount?: () => void }) {
  const calledRef = useRef(false)
  useEffect(() => {
    if (onMount && !calledRef.current) {
      calledRef.current = true
      onMount()
    }
  }, [onMount])

  return (
    <div className={cn('bg-muted text-muted-foreground flex items-center justify-center', className)}>
      {type === 'movie' ? <Film className="size-12" /> : <Tv className="size-12" />}
    </div>
  )
}

export function PosterImage({ path, url, tmdbId, tvdbId, alt, size = 'w342', type = 'movie', version, className, onAllFailed }: PosterImageProps) {
  const [loading, setLoading] = useState(true)
  const [attemptIndex, setAttemptIndex] = useState(0)
  const [prevUrlsKey, setPrevUrlsKey] = useState('')

  const artworkId = selectArtworkId(tmdbId, tvdbId)
  const artworkVersion = useArtworkStore((state) =>
    artworkId ? state.getVersion(type, artworkId, 'poster') : 0,
  )
  const urls = buildImageUrls({ artworkId, type, artworkVersion, version, url, path, size })
  const urlsKey = urls.join('|')

  if (urlsKey !== prevUrlsKey) {
    setPrevUrlsKey(urlsKey)
    setAttemptIndex(0)
    setLoading(true)
  }

  const imageUrl = urls[attemptIndex]
  if (!imageUrl) {return <FallbackIcon type={type} className={className} onMount={onAllFailed} />}

  return (
    <div className={cn('relative overflow-hidden', className)}>
      {loading ? <div className="bg-muted absolute inset-0 animate-pulse" /> : null}
      <img
        src={imageUrl}
        alt={alt}
        onLoad={() => setLoading(false)}
        onError={() => setAttemptIndex((i) => i + 1)}
        className={cn('size-full object-cover transition-opacity', loading ? 'opacity-0' : 'opacity-100')}
      />
    </div>
  )
}
