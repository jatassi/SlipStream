import { Link } from '@tanstack/react-router'
import { Clock, Eye, EyeOff } from 'lucide-react'

import { NetworkLogo } from '@/components/media/NetworkLogo'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import type { Series } from '@/types'

type SeriesCardProps = {
  series: Series
  className?: string
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}

function formatDate(dateStr: string, format: 'short' | 'medium' | 'long') {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  switch (format) {
    case 'short': {
      return `${date.getMonth() + 1}/${date.getDate()}`
    }
    case 'medium': {
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
    }
    case 'long': {
      return date.toLocaleDateString('en-US', {
        weekday: 'short',
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      })
    }
  }
}

function yearFromDate(dateStr?: string): number | null {
  if (!dateStr) {
    return null
  }
  const date = new Date(dateStr)
  return Number.isNaN(date.getTime()) ? null : date.getFullYear()
}

export function SeriesCard({
  series,
  className,
  editMode,
  selected,
  onToggleSelect,
}: SeriesCardProps) {
  const available = series.statusCounts.available + series.statusCounts.upgradable
  const total = series.statusCounts.total - series.statusCounts.unreleased
  const MonitorIcon = series.monitored ? Eye : EyeOff
  const isEnded = series.productionStatus === 'ended'
  const lastAiredYear = yearFromDate(series.lastAired)
  const firstYear = series.year || yearFromDate(series.firstAired)

  const cardContent = (
    <div className="@container relative aspect-[2/3]">
      <PosterImage
        tmdbId={series.tmdbId}
        tvdbId={series.tvdbId}
        alt={series.title}
        type="series"
        version={series.updatedAt}
        className="absolute inset-0"
      />
      {editMode ? (
        <button
          type="button"
          className="absolute top-2 left-2 z-10"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            onToggleSelect?.(series.id)
          }}
        >
          <Checkbox
            checked={selected}
            className={cn(
              'bg-background/80 size-5 border-2',
              selected && 'border-tv-500 data-[checked]:bg-tv-500',
            )}
          />
        </button>
      ) : (
        <Badge variant="secondary" className="absolute top-2 left-2 z-10 text-xs">
          {available}/{total}
        </Badge>
      )}
      <NetworkLogo
        logoUrl={series.networkLogoUrl}
        network={series.network}
        className="absolute top-2 right-2 z-10 max-w-[40%]"
      />
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-3 pt-8">
        <div className="flex items-end gap-1.5">
          <h3 className="line-clamp-2 font-semibold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
            {series.title}
          </h3>
          <MonitorIcon
            className={cn(
              'mb-0.5 size-3.5 shrink-0',
              series.monitored ? 'text-tv-400' : 'text-gray-500',
            )}
          />
        </div>
        <div className="text-sm text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">
          {isEnded ? (
            <EndedStatus firstYear={firstYear} lastAiredYear={lastAiredYear} />
          ) : (
            <ActiveStatus nextAiring={series.nextAiring} />
          )}
        </div>
      </div>
    </div>
  )

  if (editMode) {
    return (
      <button
        type="button"
        className={cn(
          'group bg-card block cursor-pointer overflow-hidden rounded-lg border-2 transition-all w-full',
          selected
            ? 'border-tv-500 glow-tv'
            : 'border-border hover:border-tv-500/50 hover:glow-tv-sm',
          className,
        )}
        onClick={() => onToggleSelect?.(series.id)}
      >
        {cardContent}
      </button>
    )
  }

  return (
    <Link
      to="/series/$id"
      params={{ id: String(series.id) }}
      className={cn(
        'group bg-card border-border hover:border-tv-500/50 hover:glow-tv block overflow-hidden rounded-lg border transition-all',
        className,
      )}
    >
      {cardContent}
    </Link>
  )
}

function EndedStatus({
  firstYear,
  lastAiredYear,
}: {
  firstYear: number | null
  lastAiredYear: number | null
}) {
  const endedLabel = `Ended ${lastAiredYear || ''}`
  const rangeLabel =
    firstYear && lastAiredYear ? `${firstYear} \u2013 ${lastAiredYear}` : endedLabel

  return (
    <>
      <span className="group-hover:hidden">{endedLabel}</span>
      <span className="hidden group-hover:inline">{rangeLabel}</span>
    </>
  )
}

function ActiveStatus({ nextAiring }: { nextAiring?: string }) {
  const shortDate = nextAiring ? formatDate(nextAiring, 'short') : null
  const mediumDate = nextAiring ? formatDate(nextAiring, 'medium') : null
  const longDate = nextAiring ? formatDate(nextAiring, 'long') : null

  return (
    <>
      <span className="group-hover:hidden">Active</span>
      {nextAiring && shortDate ? (
        <span className="hidden items-center gap-1 group-hover:inline-flex">
          <Clock className="size-3" />
          <span className="@[150px]:hidden">{shortDate}</span>
          <span className="hidden @[150px]:inline @[190px]:hidden">{mediumDate}</span>
          <span className="hidden @[190px]:inline">{longDate}</span>
        </span>
      ) : (
        <span className="hidden group-hover:inline">Active</span>
      )}
    </>
  )
}
