import { Bug, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

type DebugFooterProps = {
  developerMode: boolean
  isDebugData: boolean
  isLoadingDebugData: boolean
  isExecuting: boolean
  onLoadDebugData: () => void
}

export function DebugFooter({
  developerMode,
  isDebugData,
  isLoadingDebugData,
  isExecuting,
  onLoadDebugData,
}: DebugFooterProps) {
  return (
    <div className="flex flex-1 items-center gap-2">
      {developerMode ? (
        <DebugButton
          isLoadingDebugData={isLoadingDebugData}
          isExecuting={isExecuting}
          onLoadDebugData={onLoadDebugData}
        />
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
  )
}

type DebugButtonProps = {
  isLoadingDebugData: boolean
  isExecuting: boolean
  onLoadDebugData: () => void
}

function DebugButton({ isLoadingDebugData, isExecuting, onLoadDebugData }: DebugButtonProps) {
  const Icon = isLoadingDebugData ? Loader2 : Bug
  const iconClass = isLoadingDebugData ? 'mr-2 size-4 animate-spin' : 'mr-2 size-4'

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onLoadDebugData}
      disabled={isExecuting || isLoadingDebugData}
      className="border-orange-300 text-orange-600 hover:bg-orange-50 dark:border-orange-700 dark:text-orange-400 dark:hover:bg-orange-950/50"
    >
      <Icon className={iconClass} />
      {isLoadingDebugData ? 'Loading...' : 'Load Debug Data'}
    </Button>
  )
}
