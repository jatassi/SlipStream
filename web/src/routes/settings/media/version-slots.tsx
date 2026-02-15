import { PageHeader } from '@/components/layout/page-header'
import { VersionSlotsSection } from '@/components/settings'

import { MediaNav } from './media-nav'

export function VersionSlotsPage() {
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

      <VersionSlotsSection />
    </div>
  )
}
