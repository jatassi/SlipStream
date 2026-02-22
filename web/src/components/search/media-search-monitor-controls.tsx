import { cn } from '@/lib/utils'

import { CompletedState } from './completed-state'
import { DefaultButtons } from './default-buttons'
import { ErrorState } from './error-state'
import type { ControlState, MediaSearchMonitorControlsProps, MediaTheme, ResolvedSize } from './media-search-monitor-types'
import { ProgressState } from './progress-state'
import { SearchModal } from './search-modal'
import { SearchingState } from './searching-state'
import { useMediaSearchMonitor } from './use-media-search-monitor'

const GAP_BY_SIZE: Record<ResolvedSize, string> = {
  lg: 'gap-2',
  sm: 'gap-1.5',
  xs: 'gap-1',
}

export function MediaSearchMonitorControls(props: MediaSearchMonitorControlsProps) {
  const { theme, monitored, onMonitoredChange, monitorDisabled, className } = props
  const monitor = useMediaSearchMonitor(props)
  const isDefault = monitor.effectiveState.type === 'default'

  return (
    <div className={cn('relative', className)}>
      <div className={cn('flex items-center', GAP_BY_SIZE[monitor.size], !isDefault && 'invisible')}>
        <DefaultButtons
          size={monitor.size}
          theme={theme}
          monitored={monitored}
          monitorDisabled={monitorDisabled}
          onManualSearch={monitor.handleManualSearch}
          onAutoSearch={monitor.handleAutoSearch}
          onMonitoredChange={onMonitoredChange}
        />
      </div>

      {!isDefault && (
        <div className="absolute inset-0 flex items-center">
          <StateOverlay
            state={monitor.effectiveState}
            size={monitor.size}
            theme={theme}
            downloadProgress={monitor.downloadProgress}
            onCompletionClick={monitor.handleCompletionClick}
            onErrorDismiss={monitor.handleErrorDismiss}
          />
        </div>
      )}

      <SearchModal
        open={monitor.searchModalOpen}
        onOpenChange={monitor.handleModalClose}
        onGrabSuccess={monitor.handleGrabSuccess}
        {...monitor.searchModalProps}
      />
    </div>
  )
}

type StateOverlayProps = {
  state: ControlState
  size: ResolvedSize
  theme: MediaTheme
  downloadProgress: ReturnType<typeof useMediaSearchMonitor>['downloadProgress']
  onCompletionClick: () => void
  onErrorDismiss: () => void
}

function StateOverlay({ state, size, theme, downloadProgress, onCompletionClick, onErrorDismiss }: StateOverlayProps) {
  if (state.type === 'searching') {
    return <SearchingState size={size} theme={theme} mode={state.mode} />
  }
  if (state.type === 'progress') {
    return (
      <ProgressState
        size={size}
        theme={theme}
        progress={downloadProgress.progress}
        isPaused={downloadProgress.isPaused}
        releaseName={downloadProgress.releaseName}
        speed={downloadProgress.speed}
        eta={downloadProgress.eta}
        downloadedSize={downloadProgress.downloadedSize}
        totalSize={downloadProgress.size}
      />
    )
  }
  if (state.type === 'completed') {
    return <CompletedState size={size} theme={theme} onClick={onCompletionClick} />
  }
  if (state.type === 'error') {
    return <ErrorState size={size} theme={theme} message={state.message} onClick={onErrorDismiss} />
  }
  return null
}
