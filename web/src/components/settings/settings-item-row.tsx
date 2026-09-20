import type { ReactNode } from 'react'
import { useRef, useState } from 'react'

import { MoreHorizontal } from 'lucide-react'

import { Row } from '@/components/grouped-list'
import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'
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
      trailing={<RowTrailing title={title} detail={detail} toggle={toggle} actions={actions} />}
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
  title,
  detail,
  toggle,
  actions,
}: {
  title: string
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
      {actions === undefined ? null : <RowActions title={title} actions={actions} />}
    </div>
  )
}

function toActionItem(action: SettingsRowAction): ActionItem {
  const onClick = () => {
    void action.onClick()
  }
  if (action.confirm === undefined) {
    return { label: action.label, destructive: action.destructive, onClick }
  }
  return {
    label: action.label,
    destructive: action.destructive,
    confirm: {
      title: action.confirm.title,
      description: action.confirm.description,
      actions: [{ label: action.label, destructive: action.destructive, onClick }],
    },
  }
}

function RowActions({ title, actions }: { title: string; actions: SettingsRowAction[] }) {
  const [open, setOpen] = useState(false)
  const anchor = useRef<HTMLButtonElement>(null)

  return (
    <>
      <button
        ref={anchor}
        type="button"
        aria-label="Actions"
        onClick={() => {
          setOpen(true)
        }}
        className="press-dim focus-visible:ring-ring min-h-tap size-tap text-muted-foreground flex items-center justify-center rounded-md outline-none focus-visible:ring-[3px]"
      >
        <MoreHorizontal className="size-5" />
      </button>
      <ActionPresenter
        open={open}
        onOpenChange={setOpen}
        title={title}
        actions={actions.map((action) => toActionItem(action))}
        anchor={anchor}
      />
    </>
  )
}
