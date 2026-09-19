import type { LucideIcon } from 'lucide-react'
import { ArrowDownToLine, Ellipsis, LayoutDashboard, LayoutGrid, Search } from 'lucide-react'

import { cn } from '@/lib/utils'

import { useMobileState } from '../../shared/mobile-state-context'
import type { NativeTab } from './use-native-nav'

const TABS: { id: NativeTab; label: string; icon: LucideIcon }[] = [
  { id: 'home', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'library', label: 'Library', icon: LayoutGrid },
  { id: 'activity', label: 'Activity', icon: ArrowDownToLine },
  { id: 'search', label: 'Search', icon: Search },
  { id: 'more', label: 'More', icon: Ellipsis },
]

export function TabBar({ tab, onChange }: { tab: NativeTab; onChange: (tab: NativeTab) => void }) {
  const { queue } = useMobileState()
  const activeCount = queue.filter((q) => q.state === 'downloading').length

  return (
    <nav className="m-material m-safe-bottom absolute inset-x-0 bottom-0 z-30 shadow-[0_-1px_0_var(--material-edge)]" aria-label="Primary">
      <div className="grid h-tab-bar grid-cols-5 px-2 pt-1">
        {TABS.map(({ id, label, icon: Icon }) => {
          const active = id === tab
          return (
            <button
              key={id}
              type="button"
              onClick={() => onChange(id)}
              aria-current={active ? 'page' : undefined}
              className={cn(
                'm-press relative flex flex-col items-center justify-center gap-0.5 rounded-lg',
                active ? 'text-foreground' : 'text-muted-foreground/80',
              )}
            >
              <span className="relative">
                <Icon className="size-6" strokeWidth={active ? 2.25 : 1.75} />
                {id === 'activity' && activeCount > 0 && (
                  <span className="m-nums absolute -top-1 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-movie-500 px-1 text-[10px] leading-none font-bold text-white">
                    {activeCount}
                  </span>
                )}
              </span>
              <span className="text-[10px] leading-3 font-medium tracking-[0.01em]">{label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
