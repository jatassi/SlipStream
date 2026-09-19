import { useRef, useState } from 'react'

import { MoreHorizontal } from 'lucide-react'

import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'

export type MediaDetailMenuProps = {
  mediaLabel: string
  title: string
  onEdit: () => void
  onRefresh: () => void
  onDelete: () => void
}

export function MediaDetailMenu({ mediaLabel, title, onEdit, onRefresh, onDelete }: MediaDetailMenuProps) {
  const [open, setOpen] = useState(false)
  const anchor = useRef<HTMLButtonElement>(null)
  const label = mediaLabel.toLowerCase()

  const actions: ActionItem[] = [
    { label: 'Edit', onClick: onEdit },
    { label: 'Refresh Metadata', onClick: onRefresh },
    {
      label: 'Delete',
      destructive: true,
      confirm: {
        title: `Delete ${label}?`,
        description: `"${title}" is removed from your library. This cannot be undone.`,
        actions: [{ label: 'Delete', destructive: true, onClick: onDelete }],
      },
    },
  ]

  return (
    <>
      <button
        ref={anchor}
        type="button"
        aria-label={`${mediaLabel} actions`}
        onClick={() => setOpen(true)}
        className="press focus-visible:ring-ring flex size-11 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]"
      >
        <MoreHorizontal className="size-5" />
      </button>
      <ActionPresenter
        open={open}
        onOpenChange={setOpen}
        title={title}
        actions={actions}
        anchor={anchor}
      />
    </>
  )
}
