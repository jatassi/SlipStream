import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import type { SeasonResult } from '@/types'

const isFutureDate = (dateStr?: string) => {
  if (!dateStr) {
    return false
  }
  return new Date(dateStr) > new Date()
}

export function SeasonsList({ seasons }: { seasons: SeasonResult[] }) {
  const regularSeasons = seasons.filter((s) => s.seasonNumber > 0)

  if (regularSeasons.length === 0) {
    return null
  }

  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">
        {regularSeasons.length} {regularSeasons.length === 1 ? 'Season' : 'Seasons'}
      </h3>
      <Accordion>
        {regularSeasons.map((season) => (
          <SeasonItem key={season.seasonNumber} season={season} />
        ))}
      </Accordion>
    </div>
  )
}

function SeasonItem({ season }: { season: SeasonResult }) {
  const firstEpisode = season.episodes?.[0]
  const seasonIsFuture = firstEpisode ? isFutureDate(firstEpisode.airDate) : false

  return (
    <AccordionItem value={`season-${season.seasonNumber}`}>
      <AccordionTrigger>
        <div className="flex items-center gap-2">
          <span className={seasonIsFuture ? 'text-muted-foreground' : ''}>
            {season.name || `Season ${season.seasonNumber}`}
          </span>
          {season.episodes ? (
            <Badge variant="secondary" className="text-xs">
              {season.episodes.length} episodes
            </Badge>
          ) : null}
        </div>
      </AccordionTrigger>
      <AccordionContent>
        {season.overview ? (
          <p className="text-muted-foreground mb-2 text-sm">{season.overview}</p>
        ) : null}
        {season.episodes && season.episodes.length > 0 ? (
          <div className="space-y-1">
            {season.episodes.map((ep) => {
              const isFuture = isFutureDate(ep.airDate)
              return (
                <div
                  key={ep.episodeNumber}
                  className={`flex items-baseline gap-2 text-sm ${isFuture ? 'text-muted-foreground' : ''}`}
                >
                  <span className={isFuture ? '' : 'text-muted-foreground'}>
                    E{ep.episodeNumber}
                  </span>
                  <span className="truncate">{ep.title}</span>
                  {ep.airDate ? (
                    <span className="text-muted-foreground ml-auto shrink-0 text-xs">
                      {ep.airDate}
                    </span>
                  ) : null}
                </div>
              )
            })}
          </div>
        ) : null}
      </AccordionContent>
    </AccordionItem>
  )
}
