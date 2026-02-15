import { PageHeader } from '@/components/layout/page-header'
import { RssSyncSection } from '@/components/settings'

import { DownloadsNav } from './downloads-nav'

export function RssSyncPage() {
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

      <RssSyncSection />
    </div>
  )
}
