import { SeasonList } from '@/components/series/season-list'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { Episode, Season, Slot } from '@/types'

type SeriesSeasonsSectionProps = {
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId: number | undefined
  tmdbId: number | undefined
  imdbId: string | undefined
  seasons: Season[]
  episodes: Episode[] | undefined
  onSeasonMonitoredChange: (seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitoredChange: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot: (fileId: number, episodeId: number, slotId: number) => void
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  episodeRatings: Record<number, Record<number, number>> | undefined
}

export function SeriesSeasonsSection(props: SeriesSeasonsSectionProps) {
  return (
    <div className="space-y-6 p-6">
      <Card>
        <CardHeader>
          <CardTitle>Seasons & Episodes</CardTitle>
        </CardHeader>
        <CardContent>
          {props.seasons.length > 0 ? (
            <SeasonList
              seriesId={props.seriesId}
              seriesTitle={props.seriesTitle}
              qualityProfileId={props.qualityProfileId}
              tvdbId={props.tvdbId}
              tmdbId={props.tmdbId}
              imdbId={props.imdbId}
              seasons={props.seasons}
              episodes={props.episodes}
              onSeasonMonitoredChange={props.onSeasonMonitoredChange}
              onEpisodeMonitoredChange={props.onEpisodeMonitoredChange}
              onAssignFileToSlot={props.onAssignFileToSlot}
              isMultiVersionEnabled={props.isMultiVersionEnabled}
              enabledSlots={props.enabledSlots}
              isAssigning={props.isAssigning}
              episodeRatings={props.episodeRatings}
            />
          ) : (
            <p className="text-muted-foreground">No seasons found</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
