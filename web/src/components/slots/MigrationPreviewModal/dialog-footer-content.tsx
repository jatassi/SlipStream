import { Bug, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

type DialogFooterContentProps = {
  developerMode: boolean
  isDebugData: boolean
  isLoadingDebugData: boolean
  isLoading: boolean
  isExecuting: boolean
  hasPreview: boolean
  onLoadDebugData: () => void
  onCancel: () => void
  onExecute: () => void
}

export function DialogFooterActions({
  developerMode,
  isDebugData,
  isLoadingDebugData,
  isLoading,
  isExecuting,
  hasPreview,
  onLoadDebugData,
  onCancel,
  onExecute,
}: DialogFooterContentProps) {
  return (
    <>
      <div className="flex flex-1 items-center gap-2">
        {developerMode ? (
          <DebugButton isLoading={isLoadingDebugData} isExecuting={isExecuting} onClick={onLoadDebugData} />
        ) : null}
        {isDebugData ? (
          <Badge
            variant="outline"
            className="border-orange-300 text-orange-600 dark:text-orange-400"
          >
            Debug Mode
          </Badge>
        ) : null}
      </div>
      <Button variant="outline" onClick={onCancel} disabled={isExecuting}>
        Cancel
      </Button>
      <Button
        onClick={onExecute}
        disabled={isLoading || isExecuting || !hasPreview || isDebugData}
      >
        {isExecuting ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
        {isExecuting ? 'Executing...' : 'Execute Migration'}
      </Button>
    </>
  )
}

type DebugButtonProps = {
  isLoading: boolean
  isExecuting: boolean
  onClick: () => void
}

function DebugButton({ isLoading, isExecuting, onClick }: DebugButtonProps) {
  const Icon = isLoading ? Loader2 : Bug
  const iconClass = isLoading ? 'mr-2 size-4 animate-spin' : 'mr-2 size-4'

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onClick}
      disabled={isExecuting || isLoading}
      className="border-orange-300 text-orange-600 hover:bg-orange-50 dark:border-orange-700 dark:text-orange-400 dark:hover:bg-orange-950/50"
    >
      <Icon className={iconClass} />
      {isLoading ? 'Loading...' : 'Load Debug Data'}
    </Button>
  )
}
