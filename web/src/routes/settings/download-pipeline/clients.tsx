import { PageHeader } from '@/components/layout/page-header'
import { DownloadClientsSection } from '@/components/settings'

import { DownloadPipelineNav } from './download-pipeline-nav'

export function DownloadClientsPage() {
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

      <DownloadPipelineNav />

      <DownloadClientsSection />
    </div>
  )
}
