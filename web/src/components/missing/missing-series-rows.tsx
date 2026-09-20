import { toast } from 'sonner'

import { MediaSearchMonitorControls } from '@/components/search'
import { useUpdateSeries } from '@/hooks'
import type { MissingSeries, UpgradableSeries } from '@/types/missing'

import { MissingRow } from './missing-row'

type ToggleMonitored = (id: number, title: string, monitored: boolean) => void

function useToggleMonitored(): { toggle: ToggleMonitored; isPending: boolean } {
  const mutation = useUpdateSeries()
  const toggle: ToggleMonitored = (id, title, monitored) => {
    void (async () => {
      try {
        await mutation.mutateAsync({ id, data: { monitored } })
        toast.success(monitored ? `"${title}" monitored` : `"${title}" unmonitored`)
      } catch {
        toast.error(`Failed to update "${title}"`)
      }
    })()
  }
  return { toggle, isPending: mutation.isPending }
}

function episodeCount(count: number, word: string): string {
  return `${count} ${word} ${count === 1 ? 'episode' : 'episodes'}`
}

type SeriesLike = {
  id: number
  title: string
  year?: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  qualityProfileId: number
}

function SeriesRow({
  series,
  subtitle,
  toggle,
  isPending,
}: {
  series: SeriesLike
  subtitle: string
  toggle: ToggleMonitored
  isPending: boolean
}) {
  return (
    <MissingRow
      href={`/series/${series.id}`}
      title={series.title}
      subtitle={subtitle}
      poster={{ tmdbId: series.tmdbId, tvdbId: series.tvdbId, type: 'series' }}
    >
      <MediaSearchMonitorControls
        mediaType="series"
        seriesId={series.id}
        title={series.title}
        theme="tv"
        monitored
        onMonitoredChange={(m) => {
          toggle(series.id, series.title, m)
        }}
        monitorDisabled={isPending}
        qualityProfileId={series.qualityProfileId}
        tvdbId={series.tvdbId}
        tmdbId={series.tmdbId}
        imdbId={series.imdbId}
      />
    </MissingRow>
  )
}

export function MissingSeriesRows({ series }: { series: MissingSeries[] }) {
  const { toggle, isPending } = useToggleMonitored()

  return (
    <>
      {series.map((item) => (
        <SeriesRow
          key={item.id}
          series={item}
          subtitle={episodeCount(item.missingCount, 'missing')}
          toggle={toggle}
          isPending={isPending}
        />
      ))}
    </>
  )
}

export function UpgradableSeriesRows({ series }: { series: UpgradableSeries[] }) {
  const { toggle, isPending } = useToggleMonitored()

  return (
    <>
      {series.map((item) => (
        <SeriesRow
          key={item.id}
          series={item}
          subtitle={episodeCount(item.upgradableCount, 'upgradable')}
          toggle={toggle}
          isPending={isPending}
        />
      ))}
    </>
  )
}
