import { PageHeader } from '@/components/layout/PageHeader'
import { AutoSearchSection } from '@/components/settings'
import { DownloadsNav } from './DownloadsNav'

export function AutoSearchPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Download Pipeline"
        description="Configure indexers, download clients, and automatic search"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'Download Pipeline' },
        ]}
      />

      <DownloadsNav />

      <AutoSearchSection />
    </div>
  )
}
