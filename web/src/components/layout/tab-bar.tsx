import { Link } from '@tanstack/react-router'

import { Skeleton } from '@/components/ui/skeleton'
import { useQueue } from '@/hooks'
import { cn } from '@/lib/utils'

import { TAB_ITEMS, tabHref, type TabId, useLibraryListPath } from './use-tab-nav'

function ActivityBadge({ loading, count }: { loading: boolean; count: number }) {
  if (loading) {
    return <Skeleton className="absolute -top-1 -right-2 size-4 rounded-full" />
  }
  if (count <= 0) {
    return null
  }
  return (
    <span className="nums absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-movie-500 px-1 text-[10px] leading-none font-bold text-white">
      {count}
    </span>
  )
}

function TabLink({
  id,
  label,
  icon: Icon,
  current,
  libraryPath,
  loading,
  activeCount,
}: {
  id: TabId
  label: string
  icon: (typeof TAB_ITEMS)[number]['icon']
  current: TabId
  libraryPath: string
  loading: boolean
  activeCount: number
}) {
  const active = id === current
  return (
    <Link
      to={tabHref(id, libraryPath)}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'press relative flex flex-col items-center justify-center gap-0.5 rounded-lg',
        'focus-visible:ring-ring outline-none focus-visible:ring-[3px]',
        active ? 'text-foreground' : 'text-muted-foreground/80',
      )}
    >
      <span className="relative">
        <Icon className="size-6" strokeWidth={active ? 2.25 : 1.75} />
        {id === 'activity' && <ActivityBadge loading={loading} count={activeCount} />}
      </span>
      <span className="text-[10px] leading-3 font-medium tracking-[0.01em]">{label}</span>
    </Link>
  )
}

export function TabBar({ current }: { current: TabId }) {
  const { data, isLoading } = useQueue()
  const libraryPath = useLibraryListPath()
  const activeCount = data?.items.filter((item) => item.status === 'downloading').length ?? 0

  return (
    <nav
      className="material safe-bottom absolute inset-x-0 bottom-0 z-30 shadow-[0_-1px_0_var(--material-edge)]"
      aria-label="Primary"
    >
      <div className="grid h-tab-bar grid-cols-5 px-2 pt-1">
        {TAB_ITEMS.map((item) => (
          <TabLink
            key={item.id}
            id={item.id}
            label={item.label}
            icon={item.icon}
            current={current}
            libraryPath={libraryPath}
            loading={isLoading}
            activeCount={activeCount}
          />
        ))}
      </div>
    </nav>
  )
}
