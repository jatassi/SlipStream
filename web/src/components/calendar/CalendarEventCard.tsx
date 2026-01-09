import { Link } from '@tanstack/react-router'
import { Film, Tv } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { CalendarEvent } from '@/types/calendar'

interface CalendarEventCardProps {
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

export function CalendarEventCard({ event, compact, className }: CalendarEventCardProps) {
  const href = event.mediaType === 'movie'
    ? `/movies/${event.id}`
    : `/series/${event.seriesId}`

  return (
    <Link
      to={href}
      className={cn(
        'group block rounded-lg overflow-hidden transition-all',
        'supports-[backdrop-filter]:backdrop-blur-sm',
        'border-l-4',
        eventTypeColors[event.eventType],
        statusColors[event.status],
        'hover:bg-accent/20',
        className
      )}
    >
      <div className={cn('flex gap-2 p-2', compact && 'p-1')}>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1">
            {event.mediaType === 'movie' ? (
              <Film className="size-3 shrink-0 text-muted-foreground" />
            ) : (
              <Tv className="size-3 shrink-0 text-muted-foreground" />
            )}
            <span className={cn(
              'font-medium truncate',
              compact ? 'text-xs' : 'text-sm'
            )}>
              {event.mediaType === 'episode' ? event.seriesTitle : event.title}
            </span>
          </div>

          {event.mediaType === 'episode' && !compact && (
            <p className="text-xs text-muted-foreground truncate mt-0.5">
              {event.title.startsWith('Season ') ? (
                // Full season release (Netflix-style)
                <>{event.episodeNumber} episodes</>
              ) : (
                // Individual episode
                <>S{event.seasonNumber?.toString().padStart(2, '0')}E{event.episodeNumber?.toString().padStart(2, '0')} - {event.title}</>
              )}
            </p>
          )}

          {event.mediaType === 'episode' && compact && (
            <p className="text-[10px] text-muted-foreground truncate">
              {event.title.startsWith('Season ') ? (
                // Full season release (Netflix-style)
                <>{event.episodeNumber} eps</>
              ) : (
                // Individual episode
                <>S{event.seasonNumber}E{event.episodeNumber}</>
              )}
            </p>
          )}

          <div className="flex items-center gap-1 mt-1 flex-wrap">
            <Badge
              variant="outline"
              className={cn(
                'text-[10px] h-4 px-1',
                event.eventType === 'digital' && 'border-blue-500/50 text-blue-600',
                event.eventType === 'physical' && 'border-emerald-500/50 text-emerald-600',
                event.eventType === 'airDate' && 'border-purple-500/50 text-purple-600'
              )}
            >
              {eventTypeLabels[event.eventType]}
            </Badge>

            {event.network && (
              <Badge variant="secondary" className="text-[10px] h-4 px-1">
                {event.network}
              </Badge>
            )}

            {event.earlyAccess && (
              <Badge className="text-[10px] h-4 px-1 bg-orange-500/20 text-orange-600 border-orange-500/30">
                Early
              </Badge>
            )}

            {event.status === 'available' && (
              <Badge className="text-[10px] h-4 px-1 bg-green-500/20 text-green-600 border-green-500/30">
                Downloaded
              </Badge>
            )}
          </div>
        </div>
      </div>
    </Link>
  )
}
