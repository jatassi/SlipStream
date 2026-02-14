import { PageHeader } from '@/components/layout/PageHeader'
import { IndexersSection } from '@/components/settings'

import { DownloadsNav } from './DownloadsNav'

export function IndexersPage() {
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

      <IndexersSection />
    </div>
  )
}
