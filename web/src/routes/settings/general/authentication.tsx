import { PageHeader } from '@/components/layout/page-header'
import { AuthenticationSection } from '@/components/settings'

import { GeneralNav } from './general-nav'

export function AuthenticationPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="General"
        description="Server configuration, authentication, and notification settings"
        breadcrumbs={[{ label: 'Settings', href: '/settings/media' }, { label: 'General' }]}
      />

      <GeneralNav />

      <div className="max-w-2xl">
        <AuthenticationSection />
      </div>
    </div>
  )
}
