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
import type { ExtendedMovieResult, Movie } from '@/types'

function toStr(value: number | undefined): string | undefined {
  return value === undefined ? undefined : String(value)
}

function fmtRuntime(value: number | undefined): string | undefined {
  return value === undefined ? undefined : formatRuntime(value)
}

function joinGenres(genres: string[] | undefined): string | undefined {
  return genres && genres.length > 0 ? genres.join(', ') : undefined
}

function fmtDate(value: string | undefined): string | undefined {
  return value ? formatDate(value) : undefined
}

function deriveValues(movie: Movie, extendedData?: ExtendedMovieResult) {
  return {
    year: toStr(movie.year),
    runtime: fmtRuntime(movie.runtime),
    director: extendedData?.credits?.directors?.[0]?.name,
    genres: joinGenres(extendedData?.genres),
    addedBy: movie.addedByUsername,
    addedAt: fmtDate(movie.addedAt),
  }
}

const ITEM_CONFIG: { key: keyof ReturnType<typeof deriveValues>; icon: typeof Calendar }[] = [
  { key: 'year', icon: Calendar },
  { key: 'runtime', icon: Clock },
  { key: 'director', icon: UserStar },
  { key: 'genres', icon: Drama },
  { key: 'addedBy', icon: UserRoundPlus },
  { key: 'addedAt', icon: CalendarPlus },
]

const EXTENDED_KEYS = new Set(['director', 'genres'])

function buildMetadataItems(movie: Movie, extendedData?: ExtendedMovieResult) {
  const values = deriveValues(movie, extendedData)
  const items: { key: string; icon: typeof Calendar; value: string }[] = []
  for (const { key, icon } of ITEM_CONFIG) {
    const value = values[key]
    if (value !== undefined) {
      items.push({ key, icon, value })
    }
  }
  return items
}

type MetadataRowProps = {
  movie: Movie
  extendedData?: ExtendedMovieResult
  isExtendedDataLoading?: boolean
}

export function MetadataRow({ movie, extendedData, isExtendedDataLoading }: MetadataRowProps) {
  const items = buildMetadataItems(movie, extendedData)

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-300">
      {movie.contentRating ? (
        <span className="shrink-0 rounded border border-gray-400 px-1.5 py-0.5 text-xs font-medium text-gray-300">
          {movie.contentRating}
        </span>
      ) : null}
      {items.map(({ key, icon: Icon, value }) => (
        <span key={key} className="flex shrink-0 items-center gap-1 whitespace-nowrap">
          <Icon className="size-4 shrink-0" />
          {value}
        </span>
      ))}
      {isExtendedDataLoading && !items.some((i) => EXTENDED_KEYS.has(i.key)) ? (
        <>
          <Skeleton className="h-4 w-20 rounded bg-white/10" />
          <Skeleton className="h-4 w-24 rounded bg-white/10" />
        </>
      ) : null}
    </div>
  )
}
