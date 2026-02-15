import { Film, TrendingUp, Tv } from 'lucide-react'

import { UpgradableMoviesList } from '@/components/missing/upgradable-movies-list'
import { UpgradableSeriesList } from '@/components/missing/upgradable-series-list'
import { TabsContent } from '@/components/ui/tabs'
import type { QualityProfile } from '@/types/quality-profile'

type UpgradableTabContentProps = {
  upgradableMovies: Parameters<typeof UpgradableMoviesList>[0]['movies']
  upgradableSeries: Parameters<typeof UpgradableSeriesList>[0]['series']
  qualityProfiles: Map<number, QualityProfile>
  upgradableMovieCount: number
  upgradableEpisodeCount: number
}

export function UpgradableTabContent({
  upgradableMovies,
  upgradableSeries,
  qualityProfiles,
  upgradableMovieCount,
  upgradableEpisodeCount,
}: UpgradableTabContentProps) {
  return (
    <>
      <TabsContent value="all" className="space-y-6">
        {upgradableMovieCount > 0 && (
          <MediaSection
            icon={<Film className="text-movie-400 size-4" />}
            label="Movies"
            count={upgradableMovieCount}
            countClass="text-movie-400"
          >
            <UpgradableMoviesList movies={upgradableMovies} qualityProfiles={qualityProfiles} />
          </MediaSection>
        )}

        {upgradableSeries.length > 0 && (
          <MediaSection
            icon={<Tv className="text-tv-400 size-4" />}
            label="Episodes"
            count={upgradableEpisodeCount}
            countClass="text-tv-400"
          >
            <UpgradableSeriesList series={upgradableSeries} qualityProfiles={qualityProfiles} />
          </MediaSection>
        )}

        {upgradableMovieCount === 0 && upgradableEpisodeCount === 0 && (
          <EmptyState
            icon={<TrendingUp className="text-muted-foreground mb-4 size-12" />}
            title="No upgradable media"
            description="All monitored media meets the quality cutoff"
          />
        )}
      </TabsContent>

      <TabsContent value="movies">
        <UpgradableMoviesList movies={upgradableMovies} qualityProfiles={qualityProfiles} />
      </TabsContent>

      <TabsContent value="series">
        <UpgradableSeriesList series={upgradableSeries} qualityProfiles={qualityProfiles} />
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
