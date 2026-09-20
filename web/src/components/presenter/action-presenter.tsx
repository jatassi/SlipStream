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

export type WideSurface = 'menu' | 'dialog'

export type ActionPresenterProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  subtitle?: string
  description?: string
  actions: ActionItem[]
  anchor?: RefObject<HTMLElement | null>
  wide?: WideSurface
  locked?: boolean
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
            disabled={action.disabled}
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
            disabled={action.disabled}
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

function DialogActions({
  actions,
  locked,
  onSelect,
}: {
  actions: ActionItem[]
  locked: boolean
  onSelect: (action: ActionItem) => void
}) {
  return (
    <AlertDialogFooter>
      <AlertDialogCancel disabled={locked}>Cancel</AlertDialogCancel>
      {actions.map((action) => (
        <AlertDialogAction
          key={action.label}
          variant={action.destructive === true ? 'destructive' : 'default'}
          disabled={action.disabled}
          onClick={() => {
            onSelect(action)
          }}
        >
          {action.label}
        </AlertDialogAction>
      ))}
    </AlertDialogFooter>
  )
}

function ActionDialog({
  open,
  onOpenChange,
  title,
  description,
  actions,
  locked,
  onSelect,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  actions: ActionItem[]
  locked: boolean
  onSelect: (action: ActionItem) => void
}) {
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (locked && !next) {
          return
        }
        onOpenChange(next)
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description === undefined ? null : (
            <AlertDialogDescription>{description}</AlertDialogDescription>
          )}
        </AlertDialogHeader>
        <DialogActions actions={actions} locked={locked} onSelect={onSelect} />
      </AlertDialogContent>
    </AlertDialog>
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
    <ActionDialog
      open
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
      title={confirm.title}
      description={confirm.description}
      actions={confirm.actions}
      locked={false}
      onSelect={onRun}
    />
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
      if (action.keepOpen !== true) {
        onOpenChange(false)
      }
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

type PhoneSurfaceProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  subtitle?: string
  description?: string
  actions: ActionItem[]
  locked: boolean
  confirm: ActionConfirm | null
  onSelect: (action: ActionItem) => void
  onRun: (action: ActionItem) => void
  onCloseConfirm: () => void
}

function PhoneSurface(props: PhoneSurfaceProps) {
  return (
    <>
      <ActionSheet
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={props.title}
        subtitle={props.subtitle}
        description={props.description}
        actions={props.actions}
        onSelect={props.onSelect}
        cancelDisabled={props.locked}
      />
      <ConfirmSheet confirm={props.confirm} onClose={props.onCloseConfirm} onRun={props.onRun} />
    </>
  )
}

function guardedOpenChange(onOpenChange: (open: boolean) => void, locked: boolean) {
  return (next: boolean) => {
    if (locked && !next) {
      return
    }
    onOpenChange(next)
  }
}

export function ActionPresenter(props: ActionPresenterProps) {
  const { open, onOpenChange, title, description, actions } = props
  const locked = props.locked ?? false
  const shell = useViewport()
  const flow = useActionFlow(onOpenChange)

  if (shell === 'phone') {
    return (
      <PhoneSurface
        {...props}
        onOpenChange={guardedOpenChange(onOpenChange, locked)}
        locked={locked}
        confirm={flow.confirm}
        onSelect={flow.select}
        onRun={flow.run}
        onCloseConfirm={flow.closeConfirm}
      />
    )
  }

  if (props.wide === 'dialog') {
    return (
      <ActionDialog
        open={open}
        onOpenChange={onOpenChange}
        title={title}
        description={description}
        actions={actions}
        locked={locked}
        onSelect={flow.select}
      />
    )
  }

  return (
    <>
      <ActionMenu
        open={open}
        onOpenChange={onOpenChange}
        actions={actions}
        anchor={props.anchor}
        onSelect={flow.select}
      />
      <ConfirmAlert confirm={flow.confirm} onClose={flow.closeConfirm} onRun={flow.run} />
    </>
  )
}
