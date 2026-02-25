import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'

import { HealthCategoryCard } from './health-category-card'
import { ProwlarrTreeCard } from './prowlarr-tree-card'
import { SystemNav } from './system-nav'
import { useHealthPage } from './use-health-page'

const PAGE_TITLE = 'System'
const PAGE_DESCRIPTION = 'Monitor system health, tasks, logs, and updates'

export function SystemHealthPage() {
  const {
    isLoading,
    error,
    isProwlarrMode,
    downloadClients,
    prowlarrItem,
    indexerItems,
    regularCategories,
  } = useHealthPage()

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />
        <SystemNav />
        <LoadingState variant="list" count={5} />
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />
        <SystemNav />
        <ErrorState title="Failed to load health status" />
      </div>
    )
  }

  const indexerSection = isProwlarrMode ? (
    <ProwlarrTreeCard prowlarrItem={prowlarrItem} indexerItems={indexerItems} />
  ) : (
    <HealthCategoryCard category="indexers" items={indexerItems} />
  )

  return (
    <div className="space-y-6">
      <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />

      <SystemNav />

      <div className="space-y-4">
        <HealthCategoryCard category="downloadClients" items={downloadClients} />
        {indexerSection}
        {regularCategories.map(({ category, items }) => (
          <HealthCategoryCard key={category} category={category} items={items} />
        ))}
      </div>
    </div>
  )
}
