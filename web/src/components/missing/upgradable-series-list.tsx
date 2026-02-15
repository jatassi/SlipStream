import { TrendingUp } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { Accordion } from '@/components/ui/accordion'
import type { UpgradableSeries } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'

import { UpgradableSeriesItem } from './upgradable-series-item'
import { useUpgradableSeriesList } from './use-upgradable-series-list'

type UpgradableSeriesListProps = {
  series: UpgradableSeries[]
  qualityProfiles: Map<number, QualityProfile>
}

export function UpgradableSeriesList({ series, qualityProfiles }: UpgradableSeriesListProps) {
  const actions = useUpgradableSeriesList()

  if (series.length === 0) {
    return (
      <EmptyState
        icon={<TrendingUp className="text-tv-400 size-8" />}
        title="No upgradable episodes"
        description="All monitored episodes meet their quality cutoff"
        className="py-8"
      />
    )
  }

  return (
    <Accordion className="space-y-2">
      {series.map((s) => (
        <UpgradableSeriesItem
          key={s.id}
          series={s}
          profile={qualityProfiles.get(s.qualityProfileId)}
          onSeriesMonitored={actions.handleSeriesMonitored}
          onSeasonMonitored={actions.handleSeasonMonitored}
          onEpisodeMonitored={actions.handleEpisodeMonitored}
          isSeriesPending={actions.isSeriesPending}
          isSeasonPending={actions.isSeasonPending}
          isEpisodePending={actions.isEpisodePending}
        />
      ))}
    </Accordion>
  )
}
