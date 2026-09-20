import type { ReactNode } from 'react'

import { Dialog as DialogPrimitive } from '@base-ui/react/dialog'

import { cn } from '@/lib/utils'

import type { ActionItem } from './types'
import { destructiveActions, safeActions } from './types'

import './action-sheet.css'

export type ActionSheetProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  subtitle?: string
  description?: string
  actions: ActionItem[]
  onSelect: (action: ActionItem) => void
  cancelLabel?: string
  cancelDisabled?: boolean
}

function SheetButton({ action, onSelect }: { action: ActionItem; onSelect: (action: ActionItem) => void }) {
  return (
    <button
      type="button"
      disabled={action.disabled}
      onClick={() => {
        onSelect(action)
      }}
      className={cn(
        'press-row text-heading block h-14 w-full font-normal tracking-normal disabled:opacity-50',
        action.destructive === true ? 'text-destructive' : 'text-tv-400',
      )}
    >
      {action.label}
    </button>
  )
}

function SheetStack({
  actions,
  onSelect,
  className,
  children,
}: {
  actions: ActionItem[]
  onSelect: (action: ActionItem) => void
  className?: string
  children?: ReactNode
}) {
  if (actions.length === 0 && children === undefined) {
    return null
  }
  return (
    <div className={cn('material-heavy rounded-card divide-border/70 divide-y overflow-hidden', className)}>
      {children}
      {actions.map((action) => (
        <SheetButton key={action.label} action={action} onSelect={onSelect} />
      ))}
    </div>
  )
}

export function ActionSheet({
  open,
  onOpenChange,
  title,
  subtitle,
  description,
  actions,
  onSelect,
  cancelLabel = 'Cancel',
  cancelDisabled = false,
}: ActionSheetProps) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Backdrop className="action-sheet-scrim" />
        <DialogPrimitive.Popup className="action-sheet">
          <SheetStack actions={safeActions(actions)} onSelect={onSelect}>
            <div className="px-4 pt-3.5 pb-3 text-center">
              <DialogPrimitive.Title className="text-footnote font-semibold">
                {title}
              </DialogPrimitive.Title>
              {subtitle === undefined ? null : (
                <div className="text-muted-foreground mt-0.5 truncate font-mono text-[11px]">
                  {subtitle}
                </div>
              )}
              {description === undefined ? null : (
                <DialogPrimitive.Description className="text-footnote text-muted-foreground mt-1">
                  {description}
                </DialogPrimitive.Description>
              )}
            </div>
          </SheetStack>
          <SheetStack actions={destructiveActions(actions)} onSelect={onSelect} className="mt-2" />
          <button
            type="button"
            disabled={cancelDisabled}
            onClick={() => {
              onOpenChange(false)
            }}
            className="press text-heading bg-card rounded-card mt-2 block h-14 w-full font-semibold tracking-normal text-tv-400 disabled:opacity-50"
          >
            {cancelLabel}
          </button>
        </DialogPrimitive.Popup>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
