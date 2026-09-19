import type { RefObject } from 'react'
import { useState } from 'react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { useViewport } from '@/hooks/use-viewport'

import { ActionSheet } from './action-sheet'
import type { ActionConfirm, ActionItem } from './types'
import { destructiveActions, safeActions } from './types'

export type ActionPresenterProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  subtitle?: string
  actions: ActionItem[]
  anchor?: RefObject<HTMLElement | null>
}

function ActionMenu({
  open,
  onOpenChange,
  actions,
  anchor,
  onSelect,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  actions: ActionItem[]
  anchor?: RefObject<HTMLElement | null>
  onSelect: (action: ActionItem) => void
}) {
  const safe = safeActions(actions)
  const destructive = destructiveActions(actions)
  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuContent anchor={anchor} align="end" className="w-auto min-w-52">
        {safe.map((action) => (
          <DropdownMenuItem
            key={action.label}
            className="min-h-tap px-3"
            onClick={() => {
              onSelect(action)
            }}
          >
            {action.label}
          </DropdownMenuItem>
        ))}
        {destructive.length > 0 && <DropdownMenuSeparator />}
        {destructive.map((action) => (
          <DropdownMenuItem
            key={action.label}
            variant="destructive"
            className="min-h-tap px-3"
            onClick={() => {
              onSelect(action)
            }}
          >
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ConfirmAlert({
  confirm,
  onClose,
  onRun,
}: {
  confirm: ActionConfirm | null
  onClose: () => void
  onRun: (action: ActionItem) => void
}) {
  if (confirm === null) {
    return null
  }
  return (
    <AlertDialog
      open
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{confirm.title}</AlertDialogTitle>
          <AlertDialogDescription>{confirm.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          {confirm.actions.map((action) => (
            <AlertDialogAction
              key={action.label}
              variant={action.destructive === true ? 'destructive' : 'default'}
              onClick={() => {
                onRun(action)
              }}
            >
              {action.label}
            </AlertDialogAction>
          ))}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function ConfirmSheet({
  confirm,
  onClose,
  onRun,
}: {
  confirm: ActionConfirm | null
  onClose: () => void
  onRun: (action: ActionItem) => void
}) {
  return (
    <ActionSheet
      open={confirm !== null}
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
      title={confirm?.title ?? ''}
      description={confirm?.description}
      actions={confirm?.actions ?? []}
      onSelect={onRun}
    />
  )
}

function useActionFlow(onOpenChange: (open: boolean) => void) {
  const [confirm, setConfirm] = useState<ActionConfirm | null>(null)

  return {
    confirm,
    select: (action: ActionItem) => {
      onOpenChange(false)
      if (action.confirm !== undefined) {
        setConfirm(action.confirm)
        return
      }
      action.onClick?.()
    },
    run: (action: ActionItem) => {
      setConfirm(null)
      action.onClick?.()
    },
    closeConfirm: () => {
      setConfirm(null)
    },
  }
}

export function ActionPresenter({
  open,
  onOpenChange,
  title,
  subtitle,
  actions,
  anchor,
}: ActionPresenterProps) {
  const shell = useViewport()
  const { confirm, select, run, closeConfirm } = useActionFlow(onOpenChange)

  if (shell === 'phone') {
    return (
      <>
        <ActionSheet
          open={open}
          onOpenChange={onOpenChange}
          title={title}
          subtitle={subtitle}
          actions={actions}
          onSelect={select}
        />
        <ConfirmSheet confirm={confirm} onClose={closeConfirm} onRun={run} />
      </>
    )
  }

  return (
    <>
      <ActionMenu
        open={open}
        onOpenChange={onOpenChange}
        actions={actions}
        anchor={anchor}
        onSelect={select}
      />
      <ConfirmAlert confirm={confirm} onClose={closeConfirm} onRun={run} />
    </>
  )
}
