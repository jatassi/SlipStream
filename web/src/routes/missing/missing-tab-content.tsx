import { Binoculars, Film, Tv } from 'lucide-react'

import { MissingMoviesList } from '@/components/missing/missing-movies-list'
import { MissingSeriesList } from '@/components/missing/missing-series-list'
import { TabsContent } from '@/components/ui/tabs'

type MissingTabContentProps = {
  missingMovies: Parameters<typeof MissingMoviesList>[0]['movies']
  missingSeries: Parameters<typeof MissingSeriesList>[0]['series']
  qualityProfileNames: Map<number, string>
  missingMovieCount: number
  missingEpisodeCount: number
}

export function MissingTabContent({
  missingMovies,
  missingSeries,
  qualityProfileNames,
  missingMovieCount,
  missingEpisodeCount,
}: MissingTabContentProps) {
  return (
    <>
      <TabsContent value="all" className="space-y-6">
        {missingMovieCount > 0 && (
          <MediaSection
            icon={<Film className="text-movie-400 size-4" />}
            label="Movies"
            count={missingMovieCount}
            countClass="text-movie-400"
          >
            <MissingMoviesList movies={missingMovies} qualityProfileNames={qualityProfileNames} />
          </MediaSection>
        )}

        {missingSeries.length > 0 && (
          <MediaSection
            icon={<Tv className="text-tv-400 size-4" />}
            label="Episodes"
            count={missingEpisodeCount}
            countClass="text-tv-400"
          >
            <MissingSeriesList series={missingSeries} qualityProfileNames={qualityProfileNames} />
          </MediaSection>
        )}

        {missingMovieCount === 0 && missingEpisodeCount === 0 && (
          <EmptyState
            icon={<Binoculars className="text-muted-foreground mb-4 size-12" />}
            title="No missing media"
            description="All monitored media that has been released has been downloaded"
          />
        )}
      </TabsContent>

      <TabsContent value="movies">
        <MissingMoviesList movies={missingMovies} qualityProfileNames={qualityProfileNames} />
      </TabsContent>

      <TabsContent value="series">
        <MissingSeriesList series={missingSeries} qualityProfileNames={qualityProfileNames} />
      </TabsContent>
    </>
  )
}

function MediaSection({
  icon,
  label,
  count,
  countClass,
  children,
}: {
  icon: React.ReactNode
  label: string
  count: number
  countClass: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-3">
      <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
        {icon}
        {label}
        <span className={countClass}>({count})</span>
      </h2>
      {children}
    </div>
  )
}

function EmptyState({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode
  title: string
  description: string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      {icon}
      <h3 className="text-lg font-medium">{title}</h3>
      <p className="text-muted-foreground mt-1">{description}</p>
    </div>
  )
}
