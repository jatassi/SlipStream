import type { ReactNode } from 'react'
import { useState } from 'react'

import { MoreHorizontal } from 'lucide-react'

import { Row } from '@/components/grouped-list'
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
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Switch } from '@/components/ui/switch'

export type SettingsRowAction = {
  label: string
  onClick: () => unknown
  destructive?: boolean
  confirm?: { title: string; description: string }
}

export type SettingsRowToggle = {
  label: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
}

type SettingsItemRowProps = {
  title: string
  leading: ReactNode
  subtitle?: ReactNode
  detail?: ReactNode
  onOpen?: () => void
  openLabel?: string
  toggle?: SettingsRowToggle
  actions?: SettingsRowAction[]
}

export function SettingsItemRow({
  title,
  leading,
  subtitle,
  detail,
  onOpen,
  openLabel,
  toggle,
  actions,
}: SettingsItemRowProps) {
  return (
    <Row
      className="relative"
      leading={leading}
      title={<RowTitle title={title} onOpen={onOpen} openLabel={openLabel} />}
      subtitle={subtitle}
      trailing={<RowTrailing detail={detail} toggle={toggle} actions={actions} />}
      chevron={onOpen !== undefined && toggle === undefined && actions === undefined}
    />
  )
}

function RowTitle({
  title,
  onOpen,
  openLabel,
}: {
  title: string
  onOpen?: () => void
  openLabel?: string
}) {
  if (onOpen === undefined) {
    return title
  }
  return (
    <>
      <button
        type="button"
        aria-label={openLabel ?? title}
        onClick={onOpen}
        className="press-row focus-visible:ring-ring absolute inset-0 outline-none focus-visible:ring-[3px]"
      />
      <span className="relative">{title}</span>
    </>
  )
}

function RowTrailing({
  detail,
  toggle,
  actions,
}: {
  detail?: ReactNode
  toggle?: SettingsRowToggle
  actions?: SettingsRowAction[]
}) {
  return (
    <div className="relative flex items-center gap-3">
      {detail === undefined ? null : <span className="text-footnote">{detail}</span>}
      {toggle === undefined ? null : (
        <Switch
          aria-label={toggle.label}
          checked={toggle.checked}
          onCheckedChange={toggle.onCheckedChange}
          disabled={toggle.disabled}
        />
      )}
      {actions === undefined ? null : <RowActions actions={actions} />}
    </div>
  )
}

function RowActions({ actions }: { actions: SettingsRowAction[] }) {
  const [confirming, setConfirming] = useState<SettingsRowAction | null>(null)

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="Actions"
          className="press-dim focus-visible:ring-ring text-muted-foreground flex size-8 items-center justify-center rounded-md outline-none focus-visible:ring-[3px]"
        >
          <MoreHorizontal className="size-5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-auto min-w-40">
          {actions.map((action) => (
            <DropdownMenuItem
              key={action.label}
              variant={action.destructive === true ? 'destructive' : 'default'}
              onClick={() => {
                runAction(action, setConfirming)
              }}
            >
              {action.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <ConfirmAction action={confirming} onClose={() => setConfirming(null)} />
    </>
  )
}

function runAction(
  action: SettingsRowAction,
  setConfirming: (action: SettingsRowAction | null) => void,
): void {
  if (action.confirm === undefined) {
    void action.onClick()
    return
  }
  setConfirming(action)
}

function ConfirmAction({
  action,
  onClose,
}: {
  action: SettingsRowAction | null
  onClose: () => void
}) {
  if (action?.confirm === undefined) {
    return null
  }
  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{action.confirm.title}</AlertDialogTitle>
          <AlertDialogDescription>{action.confirm.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant={action.destructive === true ? 'destructive' : 'default'}
            onClick={() => {
              void action.onClick()
              onClose()
            }}
          >
            {action.label}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
