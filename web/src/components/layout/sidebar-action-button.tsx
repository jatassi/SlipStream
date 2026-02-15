import { cn } from '@/lib/utils'

import type { ActionItem } from './sidebar-types'

function getVariantClassName(variant: ActionItem['variant']): string {
  if (variant === 'destructive') {
    return 'text-destructive hover:bg-destructive/10 hover:text-destructive'
  }
  if (variant === 'warning') {
    return 'text-amber-500 hover:bg-amber-500/10 hover:text-amber-500'
  }
  return 'hover:bg-accent hover:text-accent-foreground'
}

export function PopoverActionButton({
  item,
  onAction,
}: {
  item: ActionItem
  onAction?: (action: string) => void
}) {
  return (
    <button
      onClick={() => onAction?.(item.action)}
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
        getVariantClassName(item.variant),
      )}
    >
      <item.icon className="size-4" />
      <span className="flex-1 text-left">{item.title}</span>
    </button>
  )
}

export function IndentedActionButton({
  item,
  onAction,
}: {
  item: ActionItem
  onAction?: (action: string) => void
}) {
  return (
    <button
      onClick={() => onAction?.(item.action)}
      className={cn(
        'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
        'border-border ml-4 border-l pl-4',
        getVariantClassName(item.variant),
      )}
    >
      <item.icon className="size-4 shrink-0" />
      <span className="flex-1 text-left">{item.title}</span>
    </button>
  )
}
