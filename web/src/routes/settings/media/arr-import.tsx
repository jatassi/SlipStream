import { ArrImportWizard } from '@/components/arr-import'
import { PageHeader } from '@/components/layout/page-header'

import { MediaNav } from './media-nav'

export function ArrImportPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Media Management"
        description="Configure root folders, quality profiles, version slots, and file naming"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'Media Management' },
        ]}
      />

      <MediaNav />

      <div className="rounded-lg border border-border bg-card p-6">
        <ArrImportWizard />
      </div>
    </div>
  )
}
