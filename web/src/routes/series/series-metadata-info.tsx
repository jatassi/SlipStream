import {
  Calendar,
  CalendarPlus,
  Clock,
  Drama,
  UserRoundPlus,
  UserStar,
} from 'lucide-react'

import { Skeleton } from '@/components/ui/skeleton'
import { formatDate, formatRuntime } from '@/lib/formatters'
import type { ExtendedSeriesResult, Series } from '@/types'

type SeriesMetadataInfoProps = {
  series: Series
  extendedData: ExtendedSeriesResult | undefined
  isExtendedDataLoading?: boolean
}

type MetadataEntry = { icon: React.ComponentType<{ className?: string }>; label: string | number | undefined | null }

function buildMetadataItems(series: Series, extendedData: ExtendedSeriesResult | undefined): MetadataItemProps[] {
  const genreLabel = extendedData?.genres && extendedData.genres.length > 0 ? extendedData.genres.join(', ') : undefined
  const candidates: MetadataEntry[] = [
    { icon: Calendar, label: series.year },
    { icon: Clock, label: series.runtime ? formatRuntime(series.runtime) : undefined },
    { icon: UserStar, label: extendedData?.credits?.creators?.[0]?.name },
    { icon: Drama, label: genreLabel },
    { icon: UserRoundPlus, label: series.addedByUsername },
    { icon: CalendarPlus, label: series.addedAt ? formatDate(series.addedAt) : undefined },
  ]
  return candidates.filter((c): c is MetadataEntry & { label: string | number } => c.label !== null && c.label !== undefined) as MetadataItemProps[]
}

function hasExtendedItems(extendedData: ExtendedSeriesResult | undefined): boolean {
  if (!extendedData) {return false}
  return !!(extendedData.credits?.creators?.[0]?.name ?? (extendedData.genres && extendedData.genres.length > 0))
}

export function SeriesMetadataInfo({ series, extendedData, isExtendedDataLoading }: SeriesMetadataInfoProps) {
  const items = buildMetadataItems(series, extendedData)
  const showExtendedSkeletons = isExtendedDataLoading && !hasExtendedItems(extendedData)
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
      <ContentRatingBadge contentRating={extendedData?.contentRating} isLoading={isExtendedDataLoading} />
      {items.map((item) => (
        <MetadataItem key={String(item.label)} icon={item.icon} label={item.label} />
      ))}
      {showExtendedSkeletons ? (
        <>
          <Skeleton className="h-4 w-20 rounded bg-white/10" />
          <Skeleton className="h-4 w-24 rounded bg-white/10" />
        </>
      ) : null}
    </div>
  )
}

function ContentRatingBadge({ contentRating, isLoading }: { contentRating?: string; isLoading?: boolean }) {
  if (contentRating) {
    return (
      <span className="shrink-0 rounded border border-gray-400 px-1.5 py-0.5 text-xs font-medium text-gray-300">
        {contentRating}
      </span>
    )
  }
  if (isLoading) {
    return <Skeleton className="h-5 w-10 rounded bg-white/10" />
  }
  return null
}

type MetadataItemProps = {
  icon: React.ComponentType<{ className?: string }>
  label: string | number
}

function MetadataItem({ icon: Icon, label }: MetadataItemProps) {
  return (
    <span className="flex shrink-0 items-center gap-1 whitespace-nowrap">
      <Icon className="size-4 shrink-0" />
      {label}
    </span>
  )
}
