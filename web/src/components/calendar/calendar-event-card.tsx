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

const eventTypeColors: Record<string, string> = {
  digital: 'border-l-blue-500',
  physical: 'border-l-emerald-500',
  airDate: 'border-l-purple-500',
}

const statusColors: Record<string, string> = {
  available: 'bg-primary/10',
  downloading: 'bg-secondary/10',
  missing: 'bg-muted/50',
}

const eventTypeLabels: Record<string, string> = {
  digital: 'Release',
  physical: 'Bluray',
  airDate: 'Air',
}

const eventTypeBadgeColors: Record<string, string> = {
  digital: 'border-blue-500/50 text-blue-600',
  physical: 'border-emerald-500/50 text-emerald-600',
  airDate: 'border-purple-500/50 text-purple-600',
}

function EpisodeDetails({ event, compact }: { event: CalendarEvent; compact?: boolean }) {
  const isSeason = event.title.startsWith('Season ')
  if (compact) {
    return <p className="text-muted-foreground truncate text-[10px]">{isSeason ? `${event.episodeNumber} eps` : `S${event.seasonNumber}E${event.episodeNumber}`}</p>
  }
  return (
    <p className="text-muted-foreground mt-0.5 truncate text-xs">
      {isSeason ? `${event.episodeNumber} episodes` : `S${event.seasonNumber?.toString().padStart(2, '0')}E${event.episodeNumber?.toString().padStart(2, '0')} - ${event.title}`}
    </p>
  )
}

function EventBadges({ event }: { event: CalendarEvent }) {
  return (
    <div className="mt-1 flex flex-wrap items-center gap-1">
      <Badge variant="outline" className={cn('h-4 px-1 text-[10px]', eventTypeBadgeColors[event.eventType])}>
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

  return (
    <Link to={href} className={cn('group block overflow-hidden rounded-lg transition-all supports-[backdrop-filter]:backdrop-blur-sm border-l-4', eventTypeColors[event.eventType], statusColors[event.status], 'hover:bg-accent/20', className)}>
      <div className={cn('flex gap-2 p-2', compact && 'p-1')}>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1">
            {event.mediaType === 'movie' ? <Film className="text-muted-foreground size-3 shrink-0" /> : <Tv className="text-muted-foreground size-3 shrink-0" />}
            <span className={cn('truncate font-medium', compact ? 'text-xs' : 'text-sm')}>
              {event.mediaType === 'episode' ? event.seriesTitle : event.title}
            </span>
          </div>
          {event.mediaType === 'episode' && <EpisodeDetails event={event} compact={compact} />}
          <EventBadges event={event} />
        </div>
      </div>
    </Link>
  )
}
