import { useEffect, useRef, useState } from 'react'

import { useCheckForUpdate, useDeveloperMode, useInstallUpdate, useUpdateStatus } from '@/hooks'
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

type DebugConfig = {
  error?: string
  progress?: number
  countdown?: number
  showReleaseNotes?: boolean
  downloadedMB?: number
}

const DEBUG_STATE_CONFIG: Record<UpdateState, DebugConfig> = {
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

type DebugSetters = {
  setDebugState: (s: UpdateState) => void
  setDebugConfig: (c: DebugConfig) => void
  setCountdown: (n: number) => void
}

type ResolveInput = {
  debugMode: boolean
  debugConfig: DebugConfig
  updateStatus: ReturnType<typeof useUpdateStatus>['data']
  debugState: UpdateState
}

function resolveDebugValues(debugConfig: DebugConfig, debugState: UpdateState) {
  return {
    currentVersion: 'dev',
    state: debugState,
    newVersion: '2.0.0' as string | undefined,
    releaseNotes: MOCK_RELEASE_NOTES as string | undefined,
    progress: debugConfig.progress ?? 0,
    error: debugConfig.error,
    downloadedMB: debugConfig.downloadedMB ?? 0,
    totalMB: 85,
  }
}

const LIVE_DEFAULTS = {
  currentVersion: 'dev',
  state: 'idle' as UpdateState,
  newVersion: undefined as string | undefined,
  releaseNotes: undefined as string | undefined,
  progress: 0,
  error: undefined as string | undefined,
  downloadedMB: 0,
  totalMB: 0,
}

function resolveLiveValues(
  updateStatus: ReturnType<typeof useUpdateStatus>['data'],
): typeof LIVE_DEFAULTS {
  if (!updateStatus) {return LIVE_DEFAULTS}
  return {
    currentVersion: updateStatus.currentVersion,
    state: updateStatus.state,
    newVersion: updateStatus.latestRelease?.version,
    releaseNotes: updateStatus.latestRelease?.releaseNotes,
    progress: updateStatus.progress,
    error: updateStatus.error,
    downloadedMB: updateStatus.downloadedMB ?? 0,
    totalMB: updateStatus.totalMB ?? 0,
  }
}

function resolveValues({ debugMode, debugConfig, updateStatus, debugState }: ResolveInput) {
  if (debugMode) {return resolveDebugValues(debugConfig, debugState)}
  return resolveLiveValues(updateStatus)
}

function useDebugState() {
  const [debugMode, setDebugMode] = useState(false)
  const [debugState, setDebugState] = useState<UpdateState>('idle')
  const [debugConfig, setDebugConfig] = useState<DebugConfig>(DEBUG_STATE_CONFIG.idle)
  const [countdown, setCountdown] = useState(5)
  const setters: DebugSetters = { setDebugState, setDebugConfig, setCountdown }
  return { debugMode, setDebugMode, debugState, debugConfig, countdown, setCountdown, setters }
}

export function useUpdatePage() {
  const developerMode = useDeveloperMode()
  const { data: updateStatus } = useUpdateStatus()
  const checkForUpdate = useCheckForUpdate()
  const installUpdate = useInstallUpdate()
  const debug = useDebugState()

  const vals = resolveValues({
    debugMode: debug.debugMode,
    debugConfig: debug.debugConfig,
    updateStatus,
    debugState: debug.debugState,
  })

  useAutoCheck({ debugMode: debug.debugMode, updateStatus, checkForUpdate })
  useRestartCountdown({
    debugMode: debug.debugMode,
    state: vals.state,
    countdown: debug.countdown,
    setCountdown: debug.setCountdown,
  })

  const handlers = buildHandlers({
    debug,
    checkForUpdate,
    installUpdate,
    state: vals.state,
  })

  return {
    developerMode,
    ...vals,
    progress: Math.min(Math.round(vals.progress), 100),
    isChecking: checkForUpdate.isPending,
    isInstalling: installUpdate.isPending,
    showReleaseNotes: vals.state === 'update-available' && !!vals.releaseNotes,
    ...handlers,
  }
}

type HandlerOptions = {
  debug: ReturnType<typeof useDebugState>
  checkForUpdate: ReturnType<typeof useCheckForUpdate>
  installUpdate: ReturnType<typeof useInstallUpdate>
  state: UpdateState
}

function buildHandlers({ debug, checkForUpdate, installUpdate, state }: HandlerOptions) {
  const cycleDebugState = () => {
    debug.setDebugMode(true)
    const nextIndex = (UPDATE_STATES.indexOf(debug.debugState) + 1) % UPDATE_STATES.length
    const nextState = UPDATE_STATES[nextIndex]
    const config = DEBUG_STATE_CONFIG[nextState]
    debug.setters.setDebugState(nextState)
    debug.setters.setDebugConfig(config)
    debug.setters.setCountdown(config.countdown ?? 5)
  }

  const handleCheckForUpdate = () => {
    if (!debug.debugMode) {
      checkForUpdate.mutate()
      return
    }
    debug.setters.setDebugState('checking')
    setTimeout(() => {
      debug.setters.setDebugState('update-available')
      debug.setters.setDebugConfig(DEBUG_STATE_CONFIG['update-available'])
    }, 2000)
  }

  const handleDownloadUpdate = () => {
    if (!debug.debugMode) {
      installUpdate.mutate()
      return
    }
    simulateDebugDownload(debug.setters)
  }

  const handleRetry = () => {
    if (state === 'error') {handleCheckForUpdate()}
    else if (state === 'failed') {handleDownloadUpdate()}
  }

  return { cycleDebugState, handleCheckForUpdate, handleDownloadUpdate, handleRetry }
}

type AutoCheckOptions = {
  debugMode: boolean
  updateStatus: ReturnType<typeof useUpdateStatus>['data']
  checkForUpdate: ReturnType<typeof useCheckForUpdate>
}

const ACTIVE_STATES = new Set<UpdateState>(['checking', 'downloading', 'installing', 'restarting'])

function useAutoCheck({ debugMode, updateStatus, checkForUpdate }: AutoCheckOptions) {
  const hasAutoChecked = useRef(false)
  useEffect(() => {
    if (hasAutoChecked.current) {return}
    if (debugMode || !updateStatus) {return}
    if (ACTIVE_STATES.has(updateStatus.state)) {return}
    hasAutoChecked.current = true
    checkForUpdate.mutate()
  }, [debugMode, updateStatus, checkForUpdate])
}

type CountdownOptions = {
  debugMode: boolean
  state: UpdateState
  countdown: number
  setCountdown: React.Dispatch<React.SetStateAction<number>>
}

function useRestartCountdown({ debugMode, state, countdown, setCountdown }: CountdownOptions) {
  useEffect(() => {
    if (debugMode || state !== 'restarting') {return}
    if (countdown === 0) {
      globalThis.location.reload()
      return
    }
    const decrement = () => setCountdown((c) => c - 1)
    const timer = setTimeout(decrement, 1000)
    return () => clearTimeout(timer)
  }, [state, countdown, debugMode, setCountdown])
}

function simulateDebugDownload({ setDebugState, setDebugConfig, setCountdown }: DebugSetters) {
  setDebugState('downloading')
  setDebugConfig({ progress: 0, downloadedMB: 0 })
  let p = 0
  const interval = setInterval(() => {
    p += Math.random() * 15
    if (p < 100) {
      setDebugConfig({ progress: p, downloadedMB: Math.round((p / 100) * 85) })
      return
    }
    clearInterval(interval)
    setDebugState('installing')
    setDebugConfig(DEBUG_STATE_CONFIG.installing)
    scheduleRestartPhase({ setDebugState, setDebugConfig, setCountdown })
  }, 300)
}

function scheduleRestartPhase({ setDebugState, setDebugConfig, setCountdown }: DebugSetters) {
  setTimeout(() => {
    setDebugState('restarting')
    setDebugConfig(DEBUG_STATE_CONFIG.restarting)
    setCountdown(5)
  }, 3000)
}
