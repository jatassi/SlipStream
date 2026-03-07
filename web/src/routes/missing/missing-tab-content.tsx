import { Binoculars } from 'lucide-react'

import { MissingMoviesList } from '@/components/missing/missing-movies-list'
import { MissingSeriesList } from '@/components/missing/missing-series-list'
import { TabsContent } from '@/components/ui/tabs'
import { getModuleOrThrow } from '@/modules'

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
          <ModuleSection moduleId="movie" count={missingMovieCount}>
            <MissingMoviesList movies={missingMovies} qualityProfileNames={qualityProfileNames} />
          </ModuleSection>
        )}

        {missingSeries.length > 0 && (
          <ModuleSection moduleId="tv" label="Episodes" count={missingEpisodeCount}>
            <MissingSeriesList series={missingSeries} qualityProfileNames={qualityProfileNames} />
          </ModuleSection>
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
