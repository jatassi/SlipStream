import { Link, useRouterState } from '@tanstack/react-router'
import { ChevronDown } from 'lucide-react'

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/stores'

import { IndentedActionButton, PopoverActionButton } from './sidebar-action-button'
import { NavLink } from './sidebar-nav-link'
import type { CollapsibleNavGroup } from './sidebar-types'
import { isActionItem } from './sidebar-types'

type CollapsibleNavSectionProps = {
  group: CollapsibleNavGroup
  collapsed: boolean
  onAction?: (action: string) => void
}

function CollapsedPopoverMenu({ group, onAction }: CollapsibleNavSectionProps) {
  const router = useRouterState()
  const isAnyChildActive = group.items.some(
    (item) => !isActionItem(item) && router.location.pathname === item.href,
  )

  return (
    <Popover>
      <PopoverTrigger
        className={cn(
          'flex w-full items-center justify-center rounded-md px-2 py-2 text-sm font-medium transition-colors',
          'hover:bg-accent hover:text-accent-foreground',
          isAnyChildActive && 'bg-accent text-accent-foreground',
        )}
      >
        <group.icon className="size-5 shrink-0" />
      </PopoverTrigger>
      <PopoverContent side="right" sideOffset={8} className="w-auto min-w-[160px] p-1">
        <div className="text-muted-foreground mb-1 px-2 py-1 text-xs font-semibold">
          {group.title}
        </div>
        {group.items.map((item) =>
          isActionItem(item) ? (
            <PopoverActionButton key={item.title} item={item} onAction={onAction} />
          ) : (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                'flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
                'hover:bg-accent hover:text-accent-foreground',
                router.location.pathname === item.href &&
                  'bg-accent text-accent-foreground font-medium',
              )}
            >
              <item.icon className="size-4" />
              <span className="flex-1">{item.title}</span>
            </Link>
          ),
        )}
      </PopoverContent>
    </Popover>
  )
}

function ExpandedCollapsibleMenu({ group, onAction }: CollapsibleNavSectionProps) {
  const router = useRouterState()
  const { expandedMenus, toggleMenu } = useUIStore()

  const isExpanded = expandedMenus[group.id] ?? false
  const isAnyChildActive = group.items.some(
    (item) => !isActionItem(item) && router.location.pathname === item.href,
  )

  return (
    <Collapsible open={isExpanded} onOpenChange={() => toggleMenu(group.id)}>
      <CollapsibleTrigger
        className={cn(
          'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
          'hover:bg-accent hover:text-accent-foreground',
          isAnyChildActive && 'text-accent-foreground',
        )}
      >
        <group.icon className="size-5 shrink-0" />
        <span className="flex-1 text-left">{group.title}</span>
        <ChevronDown
          className={cn(
            'size-4 shrink-0 transition-transform duration-200',
            isExpanded && 'rotate-180',
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="data-[ending-style]:animate-collapse-up data-[starting-style]:animate-collapse-down overflow-hidden">
        <div className="mt-1 space-y-1">
          {group.items.map((item) =>
            isActionItem(item) ? (
              <IndentedActionButton key={item.title} item={item} onAction={onAction} />
            ) : (
              <NavLink key={item.href} item={item} collapsed={false} indented />
            ),
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function CollapsibleNavSection(props: CollapsibleNavSectionProps) {
  if (props.collapsed) {
    return <CollapsedPopoverMenu {...props} />
  }
  return <ExpandedCollapsibleMenu {...props} />
}
