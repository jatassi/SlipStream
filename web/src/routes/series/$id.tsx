import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { SeasonList } from '@/components/series/season-list'
import { SeriesEditDialog } from '@/components/series/series-edit-dialog'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { SeriesActionBar } from './series-action-bar'
import { SeriesHeroSection } from './series-hero-section'
import { useSeriesDetailPage } from './use-series-detail'

function SeasonsCard({ vm }: { vm: ReturnType<typeof useSeriesDetailPage> }) {
  const { series } = vm
  if (!series) {return null}
  return (
    <Card>
      <CardHeader>
        <CardTitle>Seasons & Episodes</CardTitle>
      </CardHeader>
      <CardContent>
        {series.seasons && series.seasons.length > 0 ? (
          <SeasonList
            seriesId={series.id} seriesTitle={series.title} qualityProfileId={series.qualityProfileId}
            tvdbId={series.tvdbId} tmdbId={series.tmdbId} imdbId={series.imdbId} seasons={series.seasons}
            episodes={vm.episodes} onSeasonMonitoredChange={vm.handleSeasonMonitoredChange}
            onEpisodeMonitoredChange={vm.handleEpisodeMonitoredChange} onAssignFileToSlot={vm.handleAssignFileToSlot}
            isMultiVersionEnabled={vm.isMultiVersionEnabled} enabledSlots={vm.enabledSlots}
            isAssigning={vm.isAssigning} episodeRatings={vm.episodeRatings}
          />
        ) : (
          <p className="text-muted-foreground">No seasons found</p>
        )}
      </CardContent>
    </Card>
  )
}

export function SeriesDetailPage() {
  const vm = useSeriesDetailPage()

  if (vm.isLoading) {return <LoadingState variant="detail" />}
  if (vm.isError || !vm.series) {return <ErrorState message="Series not found" onRetry={vm.refetch} />}

  const { series } = vm
  return (
    <div className="-m-6">
      <SeriesHeroSection
        series={series} extendedData={vm.extendedData} qualityProfileName={vm.qualityProfileName}
        overviewExpanded={vm.overviewExpanded} onToggleOverview={() => vm.setOverviewExpanded(!vm.overviewExpanded)}
      />
      <SeriesActionBar
        series={series} isRefreshing={vm.isRefreshing} onToggleMonitored={vm.handleToggleMonitored}
        onRefresh={vm.handleRefresh} onEdit={() => vm.setEditDialogOpen(true)} onDelete={vm.handleDelete}
      />
      <div className="space-y-6 p-6">
        <SeasonsCard vm={vm} />
      </div>
      <SeriesEditDialog open={vm.editDialogOpen} onOpenChange={vm.setEditDialogOpen} series={series} />
    </div>
  )
}
