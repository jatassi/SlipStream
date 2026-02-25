import { PageHeader } from '@/components/layout/page-header'
import { IndexersSection } from '@/components/settings'

import { DownloadPipelineNav } from './download-pipeline-nav'

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

      <DownloadPipelineNav />

      <IndexersSection />
    </div>
  )
}
