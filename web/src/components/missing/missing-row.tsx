import type { ReactNode } from 'react'

import { Link } from '@tanstack/react-router'

import { Row } from '@/components/grouped-list'
import { PosterImage } from '@/components/media/poster-image'

type MissingRowProps = {
  href: string
  title: string
  subtitle: ReactNode
  poster: { tmdbId?: number; tvdbId?: number; type: 'movie' | 'series' }
  children: ReactNode
}

/**
 * A grouped-list row whose whole surface opens the detail while its trailing
 * search and monitor controls stay independently tappable.
 */
export function MissingRow({ href, title, subtitle, poster, children }: MissingRowProps) {
  return (
    <Row
      className="relative"
      leading={
        <PosterImage
          tmdbId={poster.tmdbId}
          tvdbId={poster.tvdbId}
          alt={title}
          type={poster.type}
          className="h-11 w-8 rounded"
        />
      }
      title={
        <>
          <Link
            to={href}
            aria-label={title}
            className="press-row focus-visible:ring-ring absolute inset-0 outline-none focus-visible:ring-[3px]"
          />
          <span className="relative">{title}</span>
        </>
      }
      subtitle={subtitle}
      trailing={
        <div role="group" aria-label={`Actions for ${title}`} className="relative flex items-center">
          {children}
        </div>
      }
    />
  )
}
