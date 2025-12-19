import { useState } from 'react'
import { cn } from '@/lib/utils'
import { BACKDROP_SIZES } from '@/lib/constants'

interface BackdropImageProps {
  path?: string | null
  alt: string
  size?: keyof typeof BACKDROP_SIZES
  className?: string
  overlay?: boolean
}

export function BackdropImage({
  path,
  alt,
  size = 'w1280',
  className,
  overlay = true,
}: BackdropImageProps) {
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)

  const imageUrl = path ? `${BACKDROP_SIZES[size]}${path}` : null

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
