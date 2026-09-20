import { toast } from 'sonner'

import { MediaSearchMonitorControls } from '@/components/search'
import { useUpdateMovie } from '@/hooks'
import type { MissingMovie, UpgradableMovie } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'
import { PREDEFINED_QUALITIES } from '@/types/quality-profile'

import { MissingRow } from './missing-row'

const qualityById = new Map(PREDEFINED_QUALITIES.map((q) => [q.id, q.name]))

type ToggleMonitored = (id: number, title: string, monitored: boolean) => void

function useToggleMonitored(): { toggle: ToggleMonitored; isPending: boolean } {
  const mutation = useUpdateMovie()
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

function subtitleParts(year: number | undefined, detail: string | undefined): string {
  return [year === undefined ? undefined : String(year), detail].filter(Boolean).join(' · ')
}

export function MissingMovieRows({
  movies,
  profileNames,
}: {
  movies: MissingMovie[]
  profileNames: Map<number, string>
}) {
  const { toggle, isPending } = useToggleMonitored()

  return (
    <>
      {movies.map((movie) => (
        <MissingRow
          key={movie.id}
          href={`/movies/${movie.id}`}
          title={movie.title}
          subtitle={subtitleParts(movie.year, profileNames.get(movie.qualityProfileId))}
          poster={{ tmdbId: movie.tmdbId, type: 'movie' }}
        >
          <MediaSearchMonitorControls
            mediaType="movie"
            movieId={movie.id}
            title={movie.title}
            theme="movie"
            monitored
            onMonitoredChange={(m) => {
              toggle(movie.id, movie.title, m)
            }}
            monitorDisabled={isPending}
            qualityProfileId={movie.qualityProfileId}
            tmdbId={movie.tmdbId}
            imdbId={movie.imdbId}
            year={movie.year}
          />
        </MissingRow>
      ))}
    </>
  )
}

function upgradeDetail(movie: UpgradableMovie, profile: QualityProfile | undefined): string {
  const current = qualityById.get(movie.currentQualityId) ?? 'Unknown'
  const cutoff = profile ? (qualityById.get(profile.cutoff) ?? 'Unknown') : 'Unknown'
  return `${current} → ${cutoff}`
}

export function UpgradableMovieRows({
  movies,
  profiles,
}: {
  movies: UpgradableMovie[]
  profiles: Map<number, QualityProfile>
}) {
  const { toggle, isPending } = useToggleMonitored()

  return (
    <>
      {movies.map((movie) => (
        <MissingRow
          key={movie.id}
          href={`/movies/${movie.id}`}
          title={movie.title}
          subtitle={subtitleParts(
            movie.year,
            upgradeDetail(movie, profiles.get(movie.qualityProfileId)),
          )}
          poster={{ tmdbId: movie.tmdbId, type: 'movie' }}
        >
          <MediaSearchMonitorControls
            mediaType="movie"
            movieId={movie.id}
            title={movie.title}
            theme="movie"
            monitored
            onMonitoredChange={(m) => {
              toggle(movie.id, movie.title, m)
            }}
            monitorDisabled={isPending}
            qualityProfileId={movie.qualityProfileId}
            tmdbId={movie.tmdbId}
            imdbId={movie.imdbId}
            year={movie.year}
          />
        </MissingRow>
      ))}
    </>
  )
}
