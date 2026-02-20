import { Link } from '@tanstack/react-router'
import { Film, Tv } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { CalendarEvent } from '@/types/calendar'

type CalendarEventCardProps = {
  event: CalendarEvent
  compact?: boolean
  className?: string
}

type MediaType = 'movie' | 'episode'

const mediaTypeStyles: Record<MediaType, { border: string; bg: string; icon: string; hover: string; badgeAccent: string }> = {
  movie: {
    border: 'border-l-movie-500',
    bg: 'bg-movie-500/10',
    icon: 'text-movie-400',
    hover: 'hover:bg-movie-500/15',
    badgeAccent: 'border-movie-500/50 text-movie-600 dark:text-movie-400',
  },
  episode: {
    border: 'border-l-tv-500',
    bg: 'bg-tv-500/10',
    icon: 'text-tv-400',
    hover: 'hover:bg-tv-500/15',
    badgeAccent: 'border-tv-500/50 text-tv-600 dark:text-tv-400',
  },
}

const eventTypeLabels: Record<string, string> = {
  digital: 'Release',
  physical: 'Bluray',
  airDate: 'Air',
}

function EpisodeDetails({ event, compact }: { event: CalendarEvent; compact?: boolean }) {
  const isSeason = event.title.startsWith('Season ')
  if (compact) {
    return <p className="text-muted-foreground truncate text-[10px]">{isSeason ? `${event.episodeNumber} eps` : `S${event.seasonNumber}E${event.episodeNumber}`}</p>
  }
  return (
    <p className="text-muted-foreground mt-0.5 truncate text-xs">
      {isSeason ? `${event.episodeNumber} episodes` : `S${event.seasonNumber.toString().padStart(2, '0')}E${event.episodeNumber.toString().padStart(2, '0')} - ${event.title}`}
    </p>
  )
}

function EventBadges({ event }: { event: CalendarEvent }) {
  const styles = mediaTypeStyles[event.mediaType]
  return (
    <div className="mt-1 flex flex-wrap items-center gap-1">
      <Badge variant="outline" className={cn('h-4 px-1 text-[10px]', styles.badgeAccent)}>
        {eventTypeLabels[event.eventType]}
      </Badge>
      {event.network ? <Badge variant="secondary" className="h-4 px-1 text-[10px]">{event.network}</Badge> : null}
      {event.earlyAccess ? <Badge className="h-4 border-orange-500/30 bg-orange-500/20 px-1 text-[10px] text-orange-600">Early</Badge> : null}
      {event.status === 'available' && <Badge className="h-4 border-green-500/30 bg-green-500/20 px-1 text-[10px] text-green-600">Downloaded</Badge>}
    </div>
  )
}

export function CalendarEventCard({ event, compact, className }: CalendarEventCardProps) {
  const href = event.mediaType === 'movie' ? `/movies/${event.id}` : `/series/${event.seriesId}`
  const styles = mediaTypeStyles[event.mediaType]

  return (
    <Link to={href} className={cn('group block overflow-hidden rounded-lg border-l-4 transition-all supports-[backdrop-filter]:backdrop-blur-sm', styles.border, styles.bg, styles.hover, className)}>
      <div className={cn('flex gap-2 p-2', compact && 'p-1')}>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1">
            {event.mediaType === 'movie' ? <Film className={cn('size-3 shrink-0', styles.icon)} /> : <Tv className={cn('size-3 shrink-0', styles.icon)} />}
            <span className={cn('truncate font-medium', compact ? 'text-xs' : 'text-sm')}>
              {event.mediaType === 'episode' ? event.seriesTitle : event.title}
            </span>
          </div>
          {event.mediaType === 'episode' && <EpisodeDetails event={event} compact={compact} />}
          {!compact && <EventBadges event={event} />}
        </div>
      </div>
    </Link>
  )
}
