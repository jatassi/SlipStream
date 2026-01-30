import { useState, useEffect, useMemo, useRef } from 'react'
import {
  CheckCircle2,
  Download,
  Loader2,
  RefreshCw,
  AlertTriangle,
  XCircle,
  ChevronDown,
  ChevronUp,
  Sparkles,
  RotateCcw,
  Bug,
} from 'lucide-react'
import { marked } from 'marked'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { useDeveloperMode, useUpdateStatus, useCheckForUpdate, useInstallUpdate } from '@/hooks'
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

const DEBUG_STATE_CONFIG: Record<UpdateState, {
  error?: string
  progress?: number
  countdown?: number
  showReleaseNotes?: boolean
  downloadedMB?: number
}> = {
  'idle': {},
  'checking': {},
  'up-to-date': {},
  'update-available': { showReleaseNotes: true },
  'error': { error: 'Connection timed out. The update server at releases.slipstream.io is not responding. Please check your firewall settings and try again.' },
  'downloading': { progress: 42, downloadedMB: 36 },
  'installing': { progress: 90 },
  'restarting': { progress: 100, countdown: 5 },
  'complete': {},
  'failed': { error: 'Installation failed: EACCES permission denied. Unable to write to /usr/local/bin/slipstream. Please run with elevated privileges or update manually.' },
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

  const displayedContent = expanded
    ? notes
    : lines.slice(0, previewLines).join('\n')

  const renderedMarkdown = useMemo(() => {
    return marked.parse(displayedContent, { async: false }) as string
  }, [displayedContent])

  return (
    <div className="space-y-3">
      <div className="text-sm font-medium text-muted-foreground">Release Notes</div>
      <div
        className={cn(
          'relative rounded-lg bg-muted/50 p-4 text-sm',
          !expanded && hasMore && 'max-h-48 overflow-hidden'
        )}
      >
        <div
          className="prose prose-sm prose-invert max-w-none
            [&_h2]:text-base [&_h2]:font-semibold [&_h2]:text-foreground [&_h2]:mt-4 [&_h2]:mb-2 [&_h2:first-child]:mt-0
            [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:text-foreground [&_h3]:mt-3 [&_h3]:mb-1.5
            [&_ul]:my-1.5 [&_ul]:pl-4 [&_li]:text-foreground/80 [&_li]:my-0.5
            [&_strong]:text-foreground [&_strong]:font-semibold
            [&_p]:text-foreground/80 [&_p]:my-1.5
            [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-xs"
          dangerouslySetInnerHTML={{ __html: renderedMarkdown }}
        />
        {!expanded && hasMore && (
          <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-muted/50 to-transparent" />
        )}
      </div>
      {hasMore && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setExpanded(!expanded)}
          className="w-full"
        >
          {expanded ? (
            <>
              <ChevronUp className="size-4 mr-1" />
              Show Less
            </>
          ) : (
            <>
              <ChevronDown className="size-4 mr-1" />
              Show More
            </>
          )}
        </Button>
      )}
    </div>
  )
}

function LogoPlaceholder() {
  return (
    <div className="flex items-center justify-center size-20 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5 border border-primary/20 shadow-lg">
      <div className="size-12 rounded-xl bg-primary flex items-center justify-center">
        <Sparkles className="size-6 text-primary-foreground" />
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
    <div className={cn(
      'absolute -bottom-1 -right-1 size-7 rounded-full flex items-center justify-center ring-4 ring-card',
      colorClasses[color]
    )}>
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
    case 'idle':
      return (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <LogoPlaceholder />
            <div>
              <div className="text-lg font-semibold">SlipStream</div>
              <div className="text-sm text-muted-foreground">Version {currentVersion}</div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <RefreshCw className="size-4 mr-2" />
              )}
              Check for Updates
            </Button>
          </div>
        </div>
      )

    case 'checking':
      return (
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <LogoPlaceholder />
            <div>
              <div className="text-lg font-semibold">SlipStream</div>
              <div className="text-sm text-muted-foreground">Version {currentVersion}</div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground mt-1">
                <Loader2 className="size-3 animate-spin" />
                <span>Checking for updates...</span>
              </div>
            </div>
          </div>
        </div>
      )

    case 'up-to-date':
      return (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="green" icon={CheckCircle2} />
            </div>
            <div>
              <div className="text-lg font-semibold text-green-500">SlipStream is Up to Date</div>
              <div className="text-sm text-muted-foreground">
                Version {currentVersion} is the latest version
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button variant="ghost" onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <RefreshCw className="size-4 mr-2" />
              )}
              Check Again
            </Button>
          </div>
        </div>
      )

    case 'update-available':
      return (
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="primary" icon={Download} />
            </div>
            <div>
              <div className="text-lg font-semibold">Update Available</div>
              <div className="text-sm text-muted-foreground">
                SlipStream {newVersion} is available
              </div>
              <div className="text-xs text-muted-foreground">
                Currently installed: {currentVersion}
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button onClick={onDownloadUpdate} disabled={isInstalling}>
              {isInstalling ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <Download className="size-4 mr-2" />
              )}
              Install Update
            </Button>
          </div>
        </div>
      )

    case 'error':
      return (
        <div className="space-y-6">
          <div className="flex items-start gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="yellow" icon={AlertTriangle} />
            </div>
            <div>
              <div className="text-lg font-semibold text-yellow-500">Unable to Check for Updates</div>
              <div className="text-sm text-muted-foreground mt-1">
                {error || 'Could not connect to update server. Please check your internet connection.'}
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button variant="outline" onClick={onRetry} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <RefreshCw className="size-4 mr-2" />
              )}
              Try Again
            </Button>
          </div>
        </div>
      )

    case 'downloading':
    case 'installing':
    case 'restarting': {
      let visualProgress = 0
      let statusText: React.ReactNode = ''

      if (state === 'downloading') {
        visualProgress = (progress ?? 0) * 0.8
        statusText = `${Math.round(downloadedMB ?? 0)}MB / ${Math.round(totalMB ?? 0)}MB`
      } else if (state === 'installing') {
        visualProgress = 80 + ((progress ?? 0) - 80) * 0.2 / 0.2
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
            <div className="text-sm text-muted-foreground">
              SlipStream {newVersion}
            </div>
            <div className="mt-2 max-w-48">
              <Progress value={Math.min(visualProgress, 100)} />
              <div className="text-xs text-muted-foreground mt-1">{statusText}</div>
            </div>
          </div>
        </div>
      )
    }

    case 'complete':
      return (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="green" icon={CheckCircle2} />
            </div>
            <div>
              <div className="text-lg font-semibold text-green-500">Update Complete</div>
              <div className="text-sm text-muted-foreground">
                Successfully updated to SlipStream {newVersion}
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button variant="ghost" onClick={onCheckForUpdate} disabled={isChecking}>
              {isChecking ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <RefreshCw className="size-4 mr-2" />
              )}
              Check for More Updates
            </Button>
          </div>
        </div>
      )

    case 'failed':
      return (
        <div className="space-y-6">
          <div className="flex items-start gap-4">
            <div className="relative">
              <LogoPlaceholder />
              <StatusBadge color="red" icon={XCircle} />
            </div>
            <div>
              <div className="text-lg font-semibold text-red-500">Update Failed</div>
              <div className="text-sm text-muted-foreground mt-1">
                {error || 'The update could not be installed. Please try again or update manually.'}
              </div>
            </div>
          </div>
          <div className="flex justify-center">
            <Button variant="outline" onClick={onRetry} disabled={isInstalling}>
              {isInstalling ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <RotateCcw className="size-4 mr-2" />
              )}
              Retry Update
            </Button>
          </div>
        </div>
      )

    default:
      return null
  }
}

export function UpdatePage() {
  const developerMode = useDeveloperMode()
  const { data: updateStatus } = useUpdateStatus()
  const checkForUpdate = useCheckForUpdate()
  const installUpdate = useInstallUpdate()

  const [debugMode, setDebugMode] = useState(false)
  const [debugState, setDebugState] = useState<UpdateState>('idle')
  const [debugConfig, setDebugConfig] = useState(DEBUG_STATE_CONFIG['idle'])
  const [countdown, setCountdown] = useState<number>(5)

  const currentVersion = updateStatus?.currentVersion ?? 'dev'
  const state = debugMode ? debugState : (updateStatus?.state ?? 'idle') as UpdateState
  const latestRelease = updateStatus?.latestRelease
  const newVersion = debugMode ? '2.0.0' : latestRelease?.version
  const releaseNotes = debugMode ? MOCK_RELEASE_NOTES : latestRelease?.releaseNotes
  const progress = debugMode ? debugConfig.progress ?? 0 : updateStatus?.progress ?? 0
  const error = debugMode ? debugConfig.error : updateStatus?.error
  const downloadedMB = debugMode ? debugConfig.downloadedMB ?? 0 : updateStatus?.downloadedMB ?? 0
  const totalMB = debugMode ? 85 : updateStatus?.totalMB ?? 0

  const hasAutoChecked = useRef(false)

  useEffect(() => {
    if (hasAutoChecked.current) return
    if (debugMode) return
    if (!updateStatus) return
    if (updateStatus.state !== 'idle') return

    hasAutoChecked.current = true
    checkForUpdate.mutate()
  }, [debugMode, updateStatus, checkForUpdate])

  useEffect(() => {
    if (debugMode) return
    if (state !== 'restarting') return
    if (countdown === 0) {
      window.location.reload()
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
          setDebugConfig(DEBUG_STATE_CONFIG['installing'])
          setTimeout(() => {
            setDebugState('restarting')
            setDebugConfig(DEBUG_STATE_CONFIG['restarting'])
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
          developerMode && (
            <Button
              variant="outline"
              size="sm"
              onClick={cycleDebugState}
              title={`Current: ${state}`}
            >
              <Bug className="size-4 mr-2" />
              Debug: {state}
            </Button>
          )
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

        {showReleaseNotes && (
          <Card className="mt-4">
            <CardContent className="py-1">
              <ReleaseNotes notes={releaseNotes} />
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
