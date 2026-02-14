import { useEffect, useMemo, useRef, useState } from 'react'

import {
  AlertTriangle,
  Bug,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Download,
  Loader2,
  RefreshCw,
  RotateCcw,
  Sparkles,
  XCircle,
} from 'lucide-react'
import { marked } from 'marked'

import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { useCheckForUpdate, useDeveloperMode, useInstallUpdate, useUpdateStatus } from '@/hooks'
import { cn } from '@/lib/utils'
import type { UpdateState } from '@/types/update'

const UPDATE_STATES: UpdateState[] = [
  'idle',
  'checking',
  'up-to-date',
  'update-available',
  'error',
  'downloading',
  'installing',
  'restarting',
  'complete',
  'failed',
]

const DEBUG_STATE_CONFIG: Record<
  UpdateState,
  {
    error?: string
    progress?: number
    countdown?: number
    showReleaseNotes?: boolean
    downloadedMB?: number
  }
> = {
  idle: {},
  checking: {},
  'up-to-date': {},
  'update-available': { showReleaseNotes: true },
  error: {
    error:
      'Connection timed out. The update server at releases.slipstream.io is not responding. Please check your firewall settings and try again.',
  },
  downloading: { progress: 42, downloadedMB: 36 },
  installing: { progress: 90 },
  restarting: { progress: 100, countdown: 5 },
  complete: {},
  failed: {
    error:
      'Installation failed: EACCES permission denied. Unable to write to /usr/local/bin/slipstream. Please run with elevated privileges or update manually.',
  },
}

const MOCK_RELEASE_NOTES = `## What's New in SlipStream 2.0.0

### Features
- **Enhanced Search**: Completely redesigned search with improved accuracy and speed
- **Smart Recommendations**: AI-powered recommendations based on your library
- **Multi-language Support**: Added support for 12 new languages
- **Dark Mode Improvements**: More consistent dark mode across all pages

### Improvements
- Faster library scanning with parallel processing
- Reduced memory usage during large imports
- Improved WebSocket stability for real-time updates
- Better handling of special characters in file names

### Bug Fixes
- Fixed issue where some episodes were not being detected correctly
- Resolved memory leak when streaming large video files
- Fixed notification delivery reliability issues
- Corrected timezone handling in calendar view

### Breaking Changes
- Minimum required Go version is now 1.21
- Database migration required (automatic on first launch)`

function ReleaseNotes({ notes }: { notes: string }) {
  const [expanded, setExpanded] = useState(false)
  const lines = notes.split('\n')
  const previewLines = 8
  const hasMore = lines.length > previewLines

  const displayedContent = expanded ? notes : lines.slice(0, previewLines).join('\n')

  const renderedMarkdown = useMemo(() => {
    return marked.parse(displayedContent, { async: false })
  }, [displayedContent])

  return (
    <div className="space-y-3">
      <div className="text-muted-foreground text-sm font-medium">Release Notes</div>
      <div
        className={cn(
          'bg-muted/50 relative rounded-lg p-4 text-sm',
          !expanded && hasMore && 'max-h-48 overflow-hidden',
        )}
      >
        <div
          className="prose prose-sm prose-invert [&_h2]:text-foreground [&_h3]:text-foreground [&_li]:text-foreground/80 [&_strong]:text-foreground [&_p]:text-foreground/80 [&_code]:bg-muted max-w-none [&_code]:rounded [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h2]:mt-4 [&_h2]:mb-2 [&_h2]:text-base [&_h2]:font-semibold [&_h2:first-child]:mt-0 [&_h3]:mt-3 [&_h3]:mb-1.5 [&_h3]:text-sm [&_h3]:font-semibold [&_li]:my-0.5 [&_p]:my-1.5 [&_strong]:font-semibold [&_ul]:my-1.5 [&_ul]:pl-4"
          dangerouslySetInnerHTML={{ __html: renderedMarkdown }}
        />
        {!expanded && hasMore ? (
          <div className="from-muted/50 absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t to-transparent" />
        ) : null}
      </div>
      {hasMore ? (
        <Button variant="ghost" size="sm" onClick={() => setExpanded(!expanded)} className="w-full">
          {expanded ? (
            <>
              <ChevronUp className="mr-1 size-4" />
              Show Less
            </>
          ) : (
            <>
              <ChevronDown className="mr-1 size-4" />
              Show More
            </>
          )}
        </Button>
      ) : null}
    </div>
  )
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

function UpdateStateDisplay({
  state,
  currentVersion,
  newVersion,
  progress,
  error,
  onCheckForUpdate,
  onDownloadUpdate,
  onRetry,
  downloadedMB,
  totalMB,
  isChecking,
  isInstalling,
}: {
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
}) {
  switch (state) {
    case 'idle': {
      return (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <LogoPlaceholder />
            <div>
              <div className="text-lg font-semibold">SlipStream</div>
              <div className="text-muted-foreground text-sm">Version {currentVersion}</div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 size-4" />
              )}
              Check for Updates
            </Button>
          </div>
        </div>
      )
    }

    case 'checking': {
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

    case 'up-to-date': {
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
          <div className="flex justify-center">
            <Button variant="ghost" onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 size-4" />
              )}
              Check Again
            </Button>
          </div>
        </div>
      )
    }

    case 'update-available': {
      return (
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="primary" icon={Download} />
            </div>
            <div>
              <div className="text-lg font-semibold">Update Available</div>
              <div className="text-muted-foreground text-sm">
                SlipStream {newVersion} is available
              </div>
              <div className="text-muted-foreground text-xs">
                Currently installed: {currentVersion}
              </div>
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

    case 'error': {
      return (
        <div className="space-y-6">
          <div className="flex items-start gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="yellow" icon={AlertTriangle} />
            </div>
            <div>
              <div className="text-lg font-semibold text-yellow-500">
                Unable to Check for Updates
              </div>
              <div className="text-muted-foreground mt-1 text-sm">
                {error ||
                  'Could not connect to update server. Please check your internet connection.'}
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button variant="outline" onClick={onRetry} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 size-4" />
              )}
              Try Again
            </Button>
          </div>
        </div>
      )
    }

    case 'downloading':
    case 'installing':
    case 'restarting': {
      let visualProgress = 0
      let statusText: React.ReactNode = ''

      if (state === 'downloading') {
        visualProgress = (progress ?? 0) * 0.8
        statusText = `${Math.round(downloadedMB ?? 0)}MB / ${Math.round(totalMB ?? 0)}MB`
      } else if (state === 'installing') {
        visualProgress = 80 + (((progress ?? 0) - 80) * 0.2) / 0.2
        statusText = 'Installing...'
      } else {
        visualProgress = 100
        statusText = (
          <span className="flex items-center gap-1.5">
            <Loader2 className="size-3 animate-spin" />
            Restarting...
          </span>
        )
      }

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

    case 'complete': {
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
          <div className="flex justify-center">
            <Button variant="ghost" onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 size-4" />
              )}
              Check for More Updates
            </Button>
          </div>
        </div>
      )
    }

    case 'failed': {
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
                {error || 'The update could not be installed. Please try again or update manually.'}
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

    default: {
      return null
    }
  }
}

export function UpdatePage() {
  const developerMode = useDeveloperMode()
  const { data: updateStatus } = useUpdateStatus()
  const checkForUpdate = useCheckForUpdate()
  const installUpdate = useInstallUpdate()

  const [debugMode, setDebugMode] = useState(false)
  const [debugState, setDebugState] = useState<UpdateState>('idle')
  const [debugConfig, setDebugConfig] = useState(DEBUG_STATE_CONFIG.idle)
  const [countdown, setCountdown] = useState<number>(5)

  const currentVersion = updateStatus?.currentVersion ?? 'dev'
  const state = debugMode ? debugState : (updateStatus?.state ?? 'idle')
  const latestRelease = updateStatus?.latestRelease
  const newVersion = debugMode ? '2.0.0' : latestRelease?.version
  const releaseNotes = debugMode ? MOCK_RELEASE_NOTES : latestRelease?.releaseNotes
  const progress = debugMode ? (debugConfig.progress ?? 0) : (updateStatus?.progress ?? 0)
  const error = debugMode ? debugConfig.error : updateStatus?.error
  const downloadedMB = debugMode
    ? (debugConfig.downloadedMB ?? 0)
    : (updateStatus?.downloadedMB ?? 0)
  const totalMB = debugMode ? 85 : (updateStatus?.totalMB ?? 0)

  const hasAutoChecked = useRef(false)

  useEffect(() => {
    if (hasAutoChecked.current) {
      return
    }
    if (debugMode) {
      return
    }
    if (!updateStatus) {
      return
    }
    if (updateStatus.state !== 'idle') {
      return
    }

    hasAutoChecked.current = true
    checkForUpdate.mutate()
  }, [debugMode, updateStatus, checkForUpdate])

  useEffect(() => {
    if (debugMode) {
      return
    }
    if (state !== 'restarting') {
      return
    }
    if (countdown === 0) {
      globalThis.location.reload()
      return
    }
    const timer = setTimeout(() => setCountdown(countdown - 1), 1000)
    return () => clearTimeout(timer)
  }, [state, countdown, debugMode])

  const cycleDebugState = () => {
    setDebugMode(true)
    const currentIndex = UPDATE_STATES.indexOf(debugState)
    const nextIndex = (currentIndex + 1) % UPDATE_STATES.length
    const nextState = UPDATE_STATES[nextIndex]
    const config = DEBUG_STATE_CONFIG[nextState]

    setDebugState(nextState)
    setDebugConfig(config)
    setCountdown(config.countdown ?? 5)
  }

  const handleCheckForUpdate = () => {
    if (debugMode) {
      setDebugState('checking')
      setTimeout(() => {
        setDebugState('update-available')
        setDebugConfig(DEBUG_STATE_CONFIG['update-available'])
      }, 2000)
    } else {
      checkForUpdate.mutate()
    }
  }

  const handleDownloadUpdate = () => {
    if (debugMode) {
      setDebugState('downloading')
      setDebugConfig({ progress: 0, downloadedMB: 0 })
      let p = 0
      const interval = setInterval(() => {
        p += Math.random() * 15
        if (p >= 100) {
          clearInterval(interval)
          setDebugState('installing')
          setDebugConfig(DEBUG_STATE_CONFIG.installing)
          setTimeout(() => {
            setDebugState('restarting')
            setDebugConfig(DEBUG_STATE_CONFIG.restarting)
            setCountdown(5)
          }, 3000)
        } else {
          setDebugConfig({ progress: p, downloadedMB: Math.round((p / 100) * 85) })
        }
      }, 300)
    } else {
      installUpdate.mutate()
    }
  }

  const handleRetry = () => {
    if (state === 'error') {
      handleCheckForUpdate()
    } else if (state === 'failed') {
      handleDownloadUpdate()
    }
  }

  const showReleaseNotes = state === 'update-available' && releaseNotes

  return (
    <div>
      <PageHeader
        title="Software Update"
        description="Check for and install SlipStream updates"
        actions={
          developerMode ? (
            <Button
              variant="outline"
              size="sm"
              onClick={cycleDebugState}
              title={`Current: ${state}`}
            >
              <Bug className="mr-2 size-4" />
              Debug: {state}
            </Button>
          ) : null
        }
      />

      <div className="max-w-lg">
        <Card>
          <CardContent className="py-1">
            <UpdateStateDisplay
              state={state}
              currentVersion={currentVersion}
              newVersion={newVersion}
              progress={Math.min(Math.round(progress), 100)}
              error={error}
              onCheckForUpdate={handleCheckForUpdate}
              onDownloadUpdate={handleDownloadUpdate}
              onRetry={handleRetry}
              downloadedMB={downloadedMB}
              totalMB={totalMB}
              isChecking={checkForUpdate.isPending}
              isInstalling={installUpdate.isPending}
            />
          </CardContent>
        </Card>

        {showReleaseNotes ? (
          <Card className="mt-4">
            <CardContent className="py-1">
              <ReleaseNotes notes={releaseNotes} />
            </CardContent>
          </Card>
        ) : null}
      </div>
    </div>
  )
}
