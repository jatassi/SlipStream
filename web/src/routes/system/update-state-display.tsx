import {
  AlertTriangle,
  CheckCircle2,
  Download,
  Loader2,
  RefreshCw,
  RotateCcw,
  Sparkles,
  XCircle,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'
import type { UpdateState } from '@/types/update'

type UpdateDisplayProps = {
  state: UpdateState
  currentVersion: string
  newVersion?: string
  progress?: number
  error?: string
  onCheckForUpdate: () => void
  onDownloadUpdate: () => void
  onRetry: () => void
  downloadedMB?: number
  totalMB?: number
  isChecking?: boolean
  isInstalling?: boolean
}

export function UpdateStateDisplay(props: UpdateDisplayProps) {
  const STATE_COMPONENTS: Partial<Record<UpdateState, React.ReactNode>> = {
    idle: <IdleState {...props} />,
    checking: <CheckingState {...props} />,
    'up-to-date': <UpToDateState {...props} />,
    'update-available': <UpdateAvailableState {...props} />,
    error: <ErrorState {...props} />,
    downloading: <ProgressState {...props} />,
    installing: <ProgressState {...props} />,
    restarting: <ProgressState {...props} />,
    complete: <CompleteState {...props} />,
    failed: <FailedState {...props} />,
  }

  return STATE_COMPONENTS[props.state] ?? null
}

function LogoPlaceholder() {
  return (
    <div className="from-primary/20 to-primary/5 border-primary/20 flex size-20 items-center justify-center rounded-2xl border bg-gradient-to-br shadow-lg">
      <div className="bg-primary flex size-12 items-center justify-center rounded-xl">
        <Sparkles className="text-primary-foreground size-6" />
      </div>
    </div>
  )
}

function StatusBadge({
  color,
  icon: Icon,
}: {
  color: 'green' | 'yellow' | 'red' | 'primary'
  icon: React.ElementType
}) {
  const colorClasses = {
    green: 'bg-green-500',
    yellow: 'bg-yellow-500',
    red: 'bg-red-500',
    primary: 'bg-primary',
  }

  return (
    <div
      className={cn(
        'ring-card absolute -right-1 -bottom-1 flex size-7 items-center justify-center rounded-full ring-4',
        colorClasses[color],
      )}
    >
      <Icon className="size-4 text-white" />
    </div>
  )
}

function CheckButton({
  onClick,
  isPending,
  label,
  variant = 'default',
}: {
  onClick: () => void
  isPending?: boolean
  label: string
  variant?: 'default' | 'ghost' | 'outline'
}) {
  return (
    <div className="flex justify-center">
      <Button variant={variant} onClick={onClick} disabled={isPending}>
        {isPending ? (
          <Loader2 className="mr-2 size-4 animate-spin" />
        ) : (
          <RefreshCw className="mr-2 size-4" />
        )}
        {label}
      </Button>
    </div>
  )
}

function IdleState({ currentVersion, onCheckForUpdate, isChecking }: UpdateDisplayProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <LogoPlaceholder />
        <div>
          <div className="text-lg font-semibold">SlipStream</div>
          <div className="text-muted-foreground text-sm">Version {currentVersion}</div>
        </div>
      </div>
      <CheckButton onClick={onCheckForUpdate} isPending={isChecking} label="Check for Updates" />
    </div>
  )
}

function CheckingState({ currentVersion }: UpdateDisplayProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <LogoPlaceholder />
        <div>
          <div className="text-lg font-semibold">SlipStream</div>
          <div className="text-muted-foreground text-sm">Version {currentVersion}</div>
          <div className="text-muted-foreground mt-1 flex items-center gap-2 text-sm">
            <Loader2 className="size-3 animate-spin" />
            <span>Checking for updates...</span>
          </div>
        </div>
      </div>
    </div>
  )
}

function UpToDateState({ currentVersion, onCheckForUpdate, isChecking }: UpdateDisplayProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <div className="relative">
          <LogoPlaceholder />
          <StatusBadge color="green" icon={CheckCircle2} />
        </div>
        <div>
          <div className="text-lg font-semibold text-green-500">SlipStream is Up to Date</div>
          <div className="text-muted-foreground text-sm">
            Version {currentVersion} is the latest version
          </div>
        </div>
      </div>
      <CheckButton
        onClick={onCheckForUpdate}
        isPending={isChecking}
        label="Check Again"
        variant="ghost"
      />
    </div>
  )
}

function UpdateAvailableState({
  newVersion,
  currentVersion,
  onDownloadUpdate,
  isInstalling,
}: UpdateDisplayProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <div className="relative">
          <LogoPlaceholder />
          <StatusBadge color="primary" icon={Download} />
        </div>
        <div>
          <div className="text-lg font-semibold">Update Available</div>
          <div className="text-muted-foreground text-sm">SlipStream {newVersion} is available</div>
          <div className="text-muted-foreground text-xs">Currently installed: {currentVersion}</div>
        </div>
      </div>
      <div className="flex justify-center">
        <Button onClick={onDownloadUpdate} disabled={isInstalling}>
          {isInstalling ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <Download className="mr-2 size-4" />
          )}
          Install Update
        </Button>
      </div>
    </div>
  )
}

function ErrorState({ error, onRetry, isChecking }: UpdateDisplayProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-start gap-4">
        <div className="relative">
          <LogoPlaceholder />
          <StatusBadge color="yellow" icon={AlertTriangle} />
        </div>
        <div>
          <div className="text-lg font-semibold text-yellow-500">Unable to Check for Updates</div>
          <div className="text-muted-foreground mt-1 text-sm">
            {error ?? 'Could not connect to update server. Please check your internet connection.'}
          </div>
        </div>
      </div>
      <CheckButton onClick={onRetry} isPending={isChecking} label="Try Again" variant="outline" />
    </div>
  )
}

type ProgressInfo = Pick<UpdateDisplayProps, 'state' | 'progress' | 'downloadedMB' | 'totalMB'>

function computeProgress({
  state,
  progress,
  downloadedMB,
  totalMB,
}: ProgressInfo): { visualProgress: number; statusText: React.ReactNode } {
  if (state === 'downloading') {
    return {
      visualProgress: (progress ?? 0) * 0.8,
      statusText: `${Math.round(downloadedMB ?? 0)}MB / ${Math.round(totalMB ?? 0)}MB`,
    }
  }
  if (state === 'installing') {
    return {
      visualProgress: 80 + (((progress ?? 0) - 80) * 0.2) / 0.2,
      statusText: 'Installing...',
    }
  }
  return {
    visualProgress: 100,
    statusText: (
      <span className="flex items-center gap-1.5">
        <Loader2 className="size-3 animate-spin" />
        Restarting...
      </span>
    ),
  }
}

function ProgressState({ state, newVersion, progress, downloadedMB, totalMB }: UpdateDisplayProps) {
  const { visualProgress, statusText } = computeProgress({ state, progress, downloadedMB, totalMB })

  return (
    <div className="flex items-center gap-4">
      <LogoPlaceholder />
      <div className="flex-1">
        <div className="text-lg font-semibold">Installing Update</div>
        <div className="text-muted-foreground text-sm">SlipStream {newVersion}</div>
        <div className="mt-2 max-w-48">
          <Progress value={Math.min(visualProgress, 100)} />
          <div className="text-muted-foreground mt-1 text-xs">{statusText}</div>
        </div>
      </div>
    </div>
  )
}

function CompleteState({ newVersion, onCheckForUpdate, isChecking }: UpdateDisplayProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <div className="relative">
          <LogoPlaceholder />
          <StatusBadge color="green" icon={CheckCircle2} />
        </div>
        <div>
          <div className="text-lg font-semibold text-green-500">Update Complete</div>
          <div className="text-muted-foreground text-sm">
            Successfully updated to SlipStream {newVersion}
          </div>
        </div>
      </div>
      <CheckButton
        onClick={onCheckForUpdate}
        isPending={isChecking}
        label="Check for More Updates"
        variant="ghost"
      />
    </div>
  )
}

function FailedState({ error, onRetry, isInstalling }: UpdateDisplayProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-start gap-4">
        <div className="relative">
          <LogoPlaceholder />
          <StatusBadge color="red" icon={XCircle} />
        </div>
        <div>
          <div className="text-lg font-semibold text-red-500">Update Failed</div>
          <div className="text-muted-foreground mt-1 text-sm">
            {error ?? 'The update could not be installed. Please try again or update manually.'}
          </div>
        </div>
      </div>
      <div className="flex justify-center">
        <Button variant="outline" onClick={onRetry} disabled={isInstalling}>
          {isInstalling ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <RotateCcw className="mr-2 size-4" />
          )}
          Retry Update
        </Button>
      </div>
    </div>
  )
}
