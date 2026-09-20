import { Group, Row } from '@/components/grouped-list'
import { formatBytes, formatDate, formatRuntime, formatStatusSummary } from '@/lib/formatters'
import type { ExtendedSeriesResult, Series, Slot } from '@/types'

export function SeriesFileGroup({
  series,
  qualityProfileName,
  isMultiVersionEnabled,
  enabledSlots,
}: {
  series: Series
  qualityProfileName?: string
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
}) {
  return (
    <Group header="File">
      <Row title="Quality Profile" trailing={qualityProfileName ?? 'Unknown'} />
      <VersionSlotsRow enabled={isMultiVersionEnabled} slots={enabledSlots} />
      <Row title="Episodes" trailing={formatStatusSummary(series.statusCounts)} />
      <Row title="Size on Disk" trailing={formatBytes(series.sizeOnDisk)} />
      <Row title="Season Folders" trailing={series.seasonFolder ? 'Yes' : 'No'} />
    </Group>
  )
}

function VersionSlotsRow({ enabled, slots }: { enabled: boolean; slots: Slot[] }) {
  if (!enabled) {
    return null
  }
  const names = slots.map((slot) => slot.name).join(', ')
  return <Row title="Version Slots" trailing={names === '' ? 'None enabled' : names} />
}

const PRODUCTION_LABEL: Record<Series['productionStatus'], string> = {
  continuing: 'Continuing',
  ended: 'Ended',
  upcoming: 'Upcoming',
}

type DetailEntry = { label: string; value?: string }

function optionalDate(value?: string): string | undefined {
  return value ? formatDate(value) : undefined
}

function seriesDetailEntries(series: Series, extended?: ExtendedSeriesResult): DetailEntry[] {
  return [
    { label: 'Status', value: PRODUCTION_LABEL[series.productionStatus] },
    { label: 'Year', value: series.year?.toString() },
    { label: 'Runtime', value: series.runtime === undefined ? undefined : formatRuntime(series.runtime) },
    { label: 'Network', value: series.network },
    { label: 'Creator', value: extended?.credits?.creators?.[0]?.name },
    { label: 'Content Rating', value: extended?.contentRating },
    { label: 'Genres', value: extended?.genres?.join(', ') },
    { label: 'First Aired', value: optionalDate(series.firstAired) },
    { label: 'Next Airing', value: optionalDate(series.nextAiring) },
    { label: 'Added', value: optionalDate(series.addedAt) },
    { label: 'Added By', value: series.addedByUsername },
  ]
}

export function SeriesDetailsGroup({
  series,
  extended,
}: {
  series: Series
  extended?: ExtendedSeriesResult
}) {
  const entries = seriesDetailEntries(series, extended).filter((entry) => entry.value !== undefined)

  return (
    <Group header="Details">
      {entries.map((entry) => (
        <Row key={entry.label} title={entry.label} trailing={entry.value} />
      ))}
      {series.path === undefined ? null : (
        <Row
          title="Path"
          trailing={<span className="max-w-44 truncate font-mono text-[12px]">{series.path}</span>}
        />
      )}
    </Group>
  )
}
