import { cn } from '@/lib/utils'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { EpisodeTable } from './EpisodeTable'
import type { Season, Episode } from '@/types'

interface SeasonListProps {
  seasons: Season[]
  episodes?: Episode[]
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
  className?: string
}

export function SeasonList({
  seasons,
  episodes = [],
  onSeasonMonitoredChange,
  className,
}: SeasonListProps) {
  // Group episodes by season
  const episodesBySeason: Record<number, Episode[]> = {}
  episodes.forEach((ep) => {
    if (!episodesBySeason[ep.seasonNumber]) {
      episodesBySeason[ep.seasonNumber] = []
    }
    episodesBySeason[ep.seasonNumber].push(ep)
  })

  // Sort seasons by number
  const sortedSeasons = [...seasons].sort((a, b) => a.seasonNumber - b.seasonNumber)

  return (
    <Accordion className={cn('space-y-2', className)}>
      {sortedSeasons.map((season) => {
        const seasonEpisodes = episodesBySeason[season.seasonNumber] || []
        const fileCount = seasonEpisodes.filter((e) => e.hasFile).length
        const totalCount = seasonEpisodes.length

        return (
          <AccordionItem
            key={season.id}
            value={`season-${season.seasonNumber}`}
            className="border rounded-lg px-4"
          >
            <AccordionTrigger className="hover:no-underline py-3">
              <div className="flex items-center gap-4 flex-1">
                <span className="font-semibold">
                  {season.seasonNumber === 0 ? 'Specials' : `Season ${season.seasonNumber}`}
                </span>
                <Badge variant={fileCount === totalCount ? 'default' : 'secondary'}>
                  {fileCount}/{totalCount}
                </Badge>
                {onSeasonMonitoredChange && (
                  <div
                    className="ml-auto mr-4"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Switch
                      checked={season.monitored}
                      onCheckedChange={(checked) =>
                        onSeasonMonitoredChange(season.seasonNumber, checked)
                      }
                    />
                  </div>
                )}
              </div>
            </AccordionTrigger>
            <AccordionContent className="pb-4">
              {seasonEpisodes.length > 0 ? (
                <EpisodeTable episodes={seasonEpisodes} />
              ) : (
                <p className="text-sm text-muted-foreground py-2">
                  No episodes found
                </p>
              )}
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}
