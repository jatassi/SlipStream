import type { LucideIcon } from 'lucide-react'
import { CheckCircle, Loader2, Search, Trash2, XCircle, Zap } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

export type RequestAction = 'approve' | 'approve-manual-search' | 'approve-auto-search' | 'deny' | 'delete'

type RequestActionsProps = {
  isPending: boolean
  isProcessing: boolean
  onAction: (action: RequestAction) => void
}

export function RequestActions(props: RequestActionsProps) {
  const approveIcon = props.isProcessing ? Loader2 : CheckCircle
  const approveIconClass = props.isProcessing ? 'animate-spin' : undefined

  return (
    <TooltipProvider>
      <div className="flex items-center gap-1">
        {props.isPending ? <>
            <ActionButton
              icon={approveIcon}
              iconClassName={approveIconClass}
              tooltip="Approve (add to library)"
              onClick={() => props.onAction('approve')}
              disabled={props.isProcessing}
            />
            <ActionButton
              icon={Search}
              tooltip="Approve & Manual Search"
              onClick={() => props.onAction('approve-manual-search')}
              disabled={props.isProcessing}
            />
            <ActionButton
              icon={Zap}
              tooltip="Approve & Auto Search"
              onClick={() => props.onAction('approve-auto-search')}
              disabled={props.isProcessing}
            />
            <ActionButton
              icon={XCircle}
              iconClassName="text-destructive"
              tooltip="Deny"
              onClick={() => props.onAction('deny')}
              disabled={props.isProcessing}
            />
          </> : null}
        <ActionButton
          icon={Trash2}
          iconClassName="text-muted-foreground hover:text-destructive"
          tooltip="Delete permanently"
          onClick={() => props.onAction('delete')}
        />
      </div>
    </TooltipProvider>
  )
}

function ActionButton({
  icon: Icon,
  iconClassName,
  tooltip,
  onClick,
  disabled,
}: {
  icon: LucideIcon
  iconClassName?: string
  tooltip: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={<Button variant="ghost" size="icon" aria-label={tooltip} onClick={onClick} disabled={disabled} />}
      >
        <Icon className={`size-4 ${iconClassName ?? ''}`} />
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  )
}
