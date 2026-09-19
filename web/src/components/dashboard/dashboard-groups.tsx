import { DownloadingGroup } from './downloading-group'
import { HealthGroup } from './health-group'
import { RecentGroup } from './recent-group'
import { StorageGroup } from './storage-group'

export function DashboardGroups() {
  return (
    <div className="stagger grid grid-cols-1 gap-4 px-screen md:grid-cols-2 [&>section]:enter-fade-up">
      <HealthGroup />
      <StorageGroup />
      <DownloadingGroup />
      <RecentGroup />
    </div>
  )
}
