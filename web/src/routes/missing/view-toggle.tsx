import { Binoculars, TrendingUp } from 'lucide-react'

import { tabsListVariants } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

import type { ViewMode } from './use-missing-page'

type ViewToggleProps = {
  isMissingView: boolean
  upgradableTotalCount: number
  isLoading: boolean
  onViewChange: (view: ViewMode) => void
}

const activeStyle =
  'bg-background text-foreground dark:border-input dark:bg-input/30 shadow-sm'
const inactiveStyle =
  'text-foreground/60 hover:text-foreground dark:text-muted-foreground dark:hover:text-foreground'
const baseStyle =
  'inline-flex h-[calc(100%-1px)] items-center justify-center gap-1.5 rounded-md border border-transparent px-1.5 py-0.5 text-sm font-medium whitespace-nowrap transition-all'

export function ViewToggle({ isMissingView, upgradableTotalCount, isLoading, onViewChange }: ViewToggleProps) {
  return (
    <div className={tabsListVariants()}>
      <button
        onClick={() => onViewChange('missing')}
        className={cn(baseStyle, isMissingView ? activeStyle : inactiveStyle)}
      >
        <Binoculars className="size-4" />
        Missing
      </button>
      <button
        onClick={() => onViewChange('upgradable')}
        className={cn(baseStyle, isMissingView ? inactiveStyle : activeStyle)}
      >
        <TrendingUp className="size-4" />
        Upgradable
        {!isLoading && upgradableTotalCount > 0 && (
          <span className="text-xs text-muted-foreground">({upgradableTotalCount})</span>
        )}
      </button>
    </div>
  )
}
