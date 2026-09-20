import { ErrorState } from '@/components/data/error-state'
import { usePushBack } from '@/components/layout/use-push-back'
import { MediaDetailHero } from '@/components/media/media-detail-hero'
import { MediaDetailMenu } from '@/components/media/media-detail-menu'
import { MediaDetailSkeleton } from '@/components/media/media-detail-skeleton'
import { MediaEditDialog } from '@/components/media/media-edit-dialog'
import { MediaOverview } from '@/components/media/media-overview'
import { aggregateMediaStatus } from '@/components/media/media-status'
import { PillRow } from '@/components/media/pill-action'
import { Screen } from '@/components/screen/screen'
import { MediaSearchMonitorControls } from '@/components/search'
import { SeasonList } from '@/components/series/season-list'
import { useUpdateSeries } from '@/hooks'
import type { Series } from '@/types'

import { SeriesDetailsGroup, SeriesFileGroup } from './series-detail-groups'
import { useSeriesDetailPage } from './use-series-detail'

type SeriesDetailVm = ReturnType<typeof useSeriesDetailPage>

const HERO_FADE = 200

function heroFacts(series: Series): string[] {
  const facts: string[] = []
  if (series.year !== undefined) {
    facts.push(String(series.year))
  }
  const seasonCount = series.seasons.filter((season) => season.seasonNumber > 0).length
  facts.push(seasonCount === 1 ? '1 Season' : `${seasonCount} Seasons`)
  return facts
}

export function SeriesDetailPage() {
  const vm = useSeriesDetailPage()
  const back = usePushBack()
  const title = vm.series?.title ?? 'Series'

  return (
    <Screen
      title={title}
      largeTitle={false}
      transparentUntil={HERO_FADE}
      back={back}
      trailing={
        vm.series === undefined ? undefined : (
          <MediaDetailMenu
            mediaLabel="Series"
            title={title}
            onEdit={() => vm.setEditDialogOpen(true)}
            onRefresh={vm.handleRefresh}
            onDelete={vm.handleDelete}
          />
        )
      }
    >
      <SeriesDetailBody vm={vm} />
    </Screen>
  )
}

function SeriesDetailBody({ vm }: { vm: SeriesDetailVm }) {
  if (vm.isLoading) {
    return <MediaDetailSkeleton />
  }
  if (vm.isError || !vm.series) {
    return <ErrorState message="Series not found" onRetry={vm.refetch} />
  }

  const { series } = vm

  return (
    <div className="mx-auto w-full max-w-3xl">
      <SeriesHeader vm={vm} series={series} />
      <MediaOverview text={series.overview} />
      <SeriesFileGroup
        series={series}
        qualityProfileName={vm.qualityProfileName}
        isMultiVersionEnabled={vm.isMultiVersionEnabled}
        enabledSlots={vm.enabledSlots}
      />
      <SeriesSeasons vm={vm} series={series} />
      <SeriesDetailsGroup series={series} extended={vm.extendedData} />
      <SeriesEditSheet vm={vm} series={series} />
    </div>
  )
}

function SeriesHeader({ vm, series }: { vm: SeriesDetailVm; series: Series }) {
  return (
    <>
      <MediaDetailHero
        kind="series"
        title={series.title}
        status={aggregateMediaStatus(series.statusCounts)}
        facts={heroFacts(series)}
        genres={vm.extendedData?.genres}
        rating={vm.extendedData?.ratings?.imdbRating}
        isMetadataLoading={vm.isExtendedDataLoading}
        tmdbId={series.tmdbId}
        tvdbId={series.tvdbId}
        version={series.updatedAt}
      />
      <PillRow className="px-screen pt-5 pb-7">
        <MediaSearchMonitorControls
          mediaType="series"
          seriesId={series.id}
          title={series.title}
          theme="tv"
          variant="pill"
          monitored={series.monitored}
          onMonitoredChange={vm.handleToggleMonitored}
          qualityProfileId={series.qualityProfileId}
          tvdbId={series.tvdbId}
          tmdbId={series.tmdbId}
          imdbId={series.imdbId}
        />
      </PillRow>
    </>
  )
}

function SeriesSeasons({ vm, series }: { vm: SeriesDetailVm; series: Series }) {
  return (
    <SeasonList
      seriesId={series.id}
      seriesTitle={series.title}
      qualityProfileId={series.qualityProfileId}
      tvdbId={series.tvdbId}
      tmdbId={series.tmdbId}
      imdbId={series.imdbId}
      seasons={series.seasons}
      episodes={vm.episodes}
      onSeasonMonitoredChange={vm.handleSeasonMonitoredChange}
      onEpisodeMonitoredChange={vm.handleEpisodeMonitoredChange}
      isMultiVersionEnabled={vm.isMultiVersionEnabled}
      enabledSlots={vm.enabledSlots}
    />
  )
}

function SeriesEditSheet({ vm, series }: { vm: SeriesDetailVm; series: Series }) {
  const updateMutation = useUpdateSeries()
  return (
    <MediaEditDialog
      open={vm.editDialogOpen}
      onOpenChange={vm.setEditDialogOpen}
      item={series}
      updateMutation={updateMutation}
      mediaLabel="Series"
      moduleType="tv"
      monitoredDescription="Search for releases and upgrade quality for all monitored episodes"
    />
  )
}
