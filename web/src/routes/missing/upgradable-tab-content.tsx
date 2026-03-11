import { TrendingUp } from 'lucide-react'

import { UpgradableMoviesList } from '@/components/missing/upgradable-movies-list'
import { UpgradableSeriesList } from '@/components/missing/upgradable-series-list'
import { TabsContent } from '@/components/ui/tabs'
import { getModuleOrThrow } from '@/modules'
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
          <ModuleSection moduleId="movie" count={upgradableMovieCount}>
            <UpgradableMoviesList movies={upgradableMovies} qualityProfiles={qualityProfiles} />
          </ModuleSection>
        )}

        {upgradableSeries.length > 0 && (
          <ModuleSection moduleId="tv" label="Episodes" count={upgradableEpisodeCount}>
            <UpgradableSeriesList series={upgradableSeries} qualityProfiles={qualityProfiles} />
          </ModuleSection>
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

const THEME_TEXT_CLASSES: Record<string, string> = {
  movie: 'text-movie-400',
  tv: 'text-tv-400',
}

function ModuleSection({
  moduleId,
  label,
  count,
  children,
}: {
  moduleId: string
  label?: string
  count: number
  children: React.ReactNode
}) {
  const mod = getModuleOrThrow(moduleId)
  const textClass = THEME_TEXT_CLASSES[mod.themeColor] ?? 'text-foreground'

  return (
    <div className="space-y-3">
      <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
        <mod.icon className={`size-4 ${textClass}`} />
        {label ?? mod.name}
        <span className={textClass}>({count})</span>
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
