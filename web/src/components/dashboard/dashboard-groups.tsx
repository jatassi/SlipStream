import { useFirstMount } from '@/hooks'
import { cn } from '@/lib/utils'

import { DownloadingGroup } from './downloading-group'
import { HealthGroup } from './health-group'
import { RecentGroup } from './recent-group'
import { StorageGroup } from './storage-group'

export function DashboardGroups() {
  const entering = useFirstMount('dashboard-groups')

  return (
    <div
      className={cn(
        'grid grid-cols-1 gap-4 px-screen md:grid-cols-2',
        entering && 'stagger [&>section]:enter-fade-up',
      )}
    >
      <HealthGroup />
      <StorageGroup />
      <DownloadingGroup />
      <RecentGroup />
    </div>
  )
}
