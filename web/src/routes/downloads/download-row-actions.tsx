import type { LucideIcon } from 'lucide-react'
import { FastForward, Pause, Play, Trash2 } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { QueueItem } from '@/types'

type ActionButtonProps = {
  icon: LucideIcon
  iconClass: string
  onClick: () => void
  disabled: boolean
  title: string
}

function ActionButton({ icon: Icon, iconClass, onClick, disabled, title }: ActionButtonProps) {
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className="group/btn"
    >
      <Icon className={iconClass} />
    </Button>
  )
}

type DownloadRowActionsProps = {
  item: QueueItem
  isMovie: boolean
  isSeries: boolean
  pauseIsPending: boolean
  resumeIsPending: boolean
  fastForwardIsPending: boolean
  onPause: () => void
  onResume: () => void
  onFastForward: () => void
  onRemove: (deleteFiles: boolean) => void
}

export function DownloadRowActions({
  item,
  isMovie,
  isSeries,
  pauseIsPending,
  resumeIsPending,
  fastForwardIsPending,
  onPause,
  onResume,
  onFastForward,
  onRemove,
}: DownloadRowActionsProps) {
  const iconClass = cn(
    'size-4 transition-all',
    isMovie && 'group-hover/btn:icon-glow-movie',
    isSeries && 'group-hover/btn:icon-glow-tv',
  )

  return (
    <div className="flex shrink-0 gap-1 self-center">
      {item.status === 'downloading' && (
        <ActionButton icon={Pause} iconClass={iconClass} onClick={onPause} disabled={pauseIsPending} title="Pause" />
      )}
      {item.status === 'paused' && (
        <ActionButton icon={Play} iconClass={iconClass} onClick={onResume} disabled={resumeIsPending} title="Resume" />
      )}
      {item.clientType === 'mock' && item.status !== 'completed' && (
        <ActionButton icon={FastForward} iconClass={iconClass} onClick={onFastForward} disabled={fastForwardIsPending} title="Fast Forward" />
      )}
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon" title="Remove" className="group/btn">
            <Trash2 className={iconClass} />
          </Button>
        }
        title="Remove download"
        description={`Are you sure you want to remove "${item.title}" from the queue?`}
        confirmLabel="Remove"
        variant="destructive"
        onConfirm={() => onRemove(false)}
      />
    </div>
  )
}
