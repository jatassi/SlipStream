import { useState } from 'react'
import { Film, Tv } from 'lucide-react'
import { cn } from '@/lib/utils'
import { POSTER_SIZES } from '@/lib/constants'

interface PosterImageProps {
  path?: string | null
  alt: string
  size?: keyof typeof POSTER_SIZES
  type?: 'movie' | 'series'
  className?: string
}

export function PosterImage({
  path,
  alt,
  size = 'w342',
  type = 'movie',
  className,
}: PosterImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)

  const imageUrl = path ? `${POSTER_SIZES[size]}${path}` : null

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
