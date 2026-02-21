import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'
import { Skeleton } from '@/components/ui/skeleton'

import { LoadingSkeleton } from './loading-skeleton'
import { MediaTabs } from './media-tabs'
import { MissingTabContent } from './missing-tab-content'
import { SearchButton } from './search-button'
import { UpgradableTabContent } from './upgradable-tab-content'
import { useMissingPage } from './use-missing-page'

const DESCRIPTIONS = {
  missing: 'Media that has been released but not yet downloaded',
  upgradable: 'Media with files below the quality cutoff',
} as const

export function MissingPage() {
  const page = useMissingPage()

  if (page.isError) {
    return (
      <div>
        <PageHeader title={page.isMissingView ? 'Missing' : 'Upgradable'} />
        <ErrorState onRetry={page.handleRefetch} />
      </div>
    )
  }

  const title = page.isMissingView ? 'Missing' : 'Upgradable'
  const description = page.isLoading ? (
    <Skeleton className="h-4 w-64" />
  ) : (
    DESCRIPTIONS[page.view]
  )

  return (
    <div>
      <PageHeader
        title={title}
        description={description}
        actions={
          <div className="flex items-center gap-2">
            <SearchButton
              isLoading={page.isLoading}
              isSearching={page.isSearching}
              searchCount={page.searchCount}
              searchButtonStyle={page.searchButtonStyle}
              onSearch={page.handleSearch}
            />
          </div>
        }
      />

      <MediaTabs
        filter={page.filter}
        onFilterChange={page.setFilter}
        isLoading={page.isLoading}
        isMissingView={page.isMissingView}
        totalCount={page.totalCount}
        movieCount={page.movieCount}
        episodeCount={page.episodeCount}
        upgradableTotalCount={page.upgradableTotalCount}
        onViewChange={page.setView}
      >
        <TabContent page={page} />
      </MediaTabs>
    </div>
  )
}

function TabContent({ page }: { page: ReturnType<typeof useMissingPage> }) {
  if (page.isLoading) {
    return <LoadingSkeleton />
  }

  if (page.isMissingView) {
    return (
      <MissingTabContent
        missingMovies={page.missingMovies}
        missingSeries={page.missingSeries}
        qualityProfileNames={page.qualityProfileNames}
        missingMovieCount={page.missingMovieCount}
        missingEpisodeCount={page.missingEpisodeCount}
      />
    )
  }

  return (
    <UpgradableTabContent
      upgradableMovies={page.upgradableMovies}
      upgradableSeries={page.upgradableSeries}
      qualityProfiles={page.qualityProfileMap}
      upgradableMovieCount={page.upgradableMovieCount}
      upgradableEpisodeCount={page.upgradableEpisodeCount}
    />
  )
}
