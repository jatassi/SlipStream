import { useState } from 'react'

import { MediaSearchMonitorControls } from '@/components/search'
import type { ControlVariant } from '@/components/search/media-search-monitor-types'
import { Badge } from '@/components/ui/badge'

import type { MediaTheme } from './controls-types'

export function LiveDefaultRow({ theme, variant }: { theme: MediaTheme; variant: ControlVariant }) {
  const [monitored, setMonitored] = useState(true)

  const commonProps = {
    theme,
    variant,
    monitored,
    onMonitoredChange: setMonitored,
    qualityProfileId: 1,
    tmdbId: 550,
  }

  return (
    <div className="flex items-center gap-4">
      <Badge variant="outline" className="w-12 justify-center text-xs">
        {variant}
      </Badge>
      {theme === 'movie' ? (
        <MediaSearchMonitorControls
          mediaType="movie"
          movieId={1}
          title="The Matrix"
          imdbId="tt0133093"
          year={1999}
          {...commonProps}
        />
      ) : (
        <MediaSearchMonitorControls
          mediaType="series"
          seriesId={1}
          title="Breaking Bad"
          tvdbId={81_189}
          imdbId="tt0903747"
          {...commonProps}
        />
      )}
      <span className="text-muted-foreground text-xs">
        {monitored ? 'monitored' : 'unmonitored'}
      </span>
    </div>
  )
}
