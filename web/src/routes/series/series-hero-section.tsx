import { SlidersVertical } from 'lucide-react'

import { BackdropImage } from '@/components/media/backdrop-image'
import { PosterImage } from '@/components/media/poster-image'
import { ProductionStatusBadge } from '@/components/media/production-status-badge'
import { StudioLogo } from '@/components/media/studio-logo'
import { TitleTreatment } from '@/components/media/title-treatment'
import { Badge } from '@/components/ui/badge'
import { formatStatusSummary } from '@/lib/formatters'
import type { ExtendedSeriesResult, Series } from '@/types'

import { SeriesMetadataInfo } from './series-metadata-info'
import { SeriesRatingsBar } from './series-ratings-bar'

type SeriesHeroSectionProps = {
  series: Series
  extendedData: ExtendedSeriesResult | undefined
  qualityProfileName: string | undefined
  overviewExpanded: boolean
  onToggleOverview: () => void
}

function HeroBadges({ series, qualityProfileName }: { series: Series; qualityProfileName: string | undefined }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <ProductionStatusBadge status={series.productionStatus} />
      <Badge variant="secondary">{formatStatusSummary(series.statusCounts)}</Badge>
      {qualityProfileName ? (
        <Badge variant="secondary" className="gap-1">
          <SlidersVertical className="size-3" />
          {qualityProfileName}
        </Badge>
      ) : null}
    </div>
  )
}

function HeroContent({ series, extendedData, qualityProfileName, overviewExpanded, onToggleOverview }: SeriesHeroSectionProps) {
  return (
    <div className="flex-1 space-y-2">
      <HeroBadges series={series} qualityProfileName={qualityProfileName} />
      <TitleTreatment tmdbId={series.tmdbId} tvdbId={series.tvdbId} type="series" alt={series.title} version={series.updatedAt} fallback={<h1 className="text-3xl font-bold text-white">{series.title}</h1>} />
      <SeriesMetadataInfo series={series} extendedData={extendedData} />
      {extendedData?.ratings ? <SeriesRatingsBar ratings={extendedData.ratings} /> : null}
      {series.overview ? (
        <button type="button" className={`max-w-2xl cursor-pointer text-sm text-gray-300 text-left ${overviewExpanded ? '' : 'line-clamp-2'}`} onClick={onToggleOverview}>
          {series.overview}
        </button>
      ) : null}
    </div>
  )
}

export function SeriesHeroSection(props: SeriesHeroSectionProps) {
  const { series } = props
  return (
    <div className="relative h-64 md:h-80">
      <BackdropImage tmdbId={series.tmdbId} tvdbId={series.tvdbId} type="series" alt={series.title} version={series.updatedAt} className="absolute inset-0" />
      {series.network ? (
        <StudioLogo
          tmdbId={series.tmdbId} type="series" alt={series.network} version={series.updatedAt} className="absolute top-4 right-4 z-10"
          fallback={<span className="rounded bg-black/50 px-2.5 py-1 text-xs font-medium text-white/80 backdrop-blur-sm">{series.network}</span>}
        />
      ) : null}
      <div className="absolute inset-0 flex items-end p-6">
        <div className="flex max-w-4xl items-end gap-6">
          <div className="hidden shrink-0 md:block">
            <PosterImage tmdbId={series.tmdbId} tvdbId={series.tvdbId} alt={series.title} type="series" version={series.updatedAt} className="h-60 w-40 rounded-lg shadow-lg" />
          </div>
          <HeroContent {...props} />
        </div>
      </div>
    </div>
  )
}
