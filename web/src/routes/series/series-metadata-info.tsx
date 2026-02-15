import {
  Calendar,
  CalendarPlus,
  Clock,
  Drama,
  UserRoundPlus,
  UserStar,
} from 'lucide-react'

import { formatDate, formatRuntime } from '@/lib/formatters'
import type { ExtendedSeriesResult, Series } from '@/types'

type SeriesMetadataInfoProps = {
  series: Series
  extendedData: ExtendedSeriesResult | undefined
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

export function SeriesMetadataInfo({ series, extendedData }: SeriesMetadataInfoProps) {
  const items = buildMetadataItems(series, extendedData)
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
      {extendedData?.contentRating ? (
        <span className="shrink-0 rounded border border-gray-400 px-1.5 py-0.5 text-xs font-medium text-gray-300">
          {extendedData.contentRating}
        </span>
      ) : null}
      {items.map((item) => (
        <MetadataItem key={String(item.label)} icon={item.icon} label={item.label} />
      ))}
    </div>
  )
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
