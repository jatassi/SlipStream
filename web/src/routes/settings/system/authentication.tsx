import { PageHeader } from '@/components/layout/page-header'
import { AuthenticationSection } from '@/components/settings'

import { SystemNav } from './system-nav'

export function AuthenticationPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="System"
        description="Server configuration and authentication settings"
        breadcrumbs={[{ label: 'Settings', href: '/settings/media' }, { label: 'System' }]}
      />

      <SystemNav />

      <div className="max-w-2xl">
        <AuthenticationSection />
      </div>
    </div>
  )
}
