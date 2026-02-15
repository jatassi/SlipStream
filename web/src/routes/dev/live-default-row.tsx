import { useState } from 'react'

import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'

import type { ControlSize, MediaTheme } from './controls-types'

export function LiveDefaultRow({ theme, size }: { theme: MediaTheme; size: ControlSize }) {
  const [monitored, setMonitored] = useState(true)

  const commonProps = {
    theme,
    size,
    monitored,
    onMonitoredChange: setMonitored,
    qualityProfileId: 1,
    tmdbId: 550,
  }

  return (
    <div className="flex items-center gap-4">
      <Badge variant="outline" className="w-8 justify-center text-xs">
        {size}
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
