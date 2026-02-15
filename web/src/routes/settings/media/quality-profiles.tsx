import { PageHeader } from '@/components/layout/page-header'
import { QualityProfilesSection } from '@/components/settings'

import { MediaNav } from './media-nav'

export function QualityProfilesPage() {
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

      <QualityProfilesSection />
    </div>
  )
}
