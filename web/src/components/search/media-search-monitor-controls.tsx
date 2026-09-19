import { cn } from '@/lib/utils'

import { CompletedState } from './completed-state'
import { DefaultButtons } from './default-buttons'
import { ErrorState } from './error-state'
import type {
  ControlState,
  ControlVariant,
  MediaSearchMonitorControlsProps,
  MediaTheme,
} from './media-search-monitor-types'
import { ProgressState } from './progress-state'
import { SearchModal } from './search-modal'
import { SearchingState } from './searching-state'
import { useMediaSearchMonitor } from './use-media-search-monitor'

const GAP: Record<ControlVariant, string> = {
  pill: 'gap-2',
  row: 'gap-1.5',
}

export function MediaSearchMonitorControls(props: MediaSearchMonitorControlsProps) {
  const { theme, monitored, onMonitoredChange, monitorDisabled, className } = props
  const monitor = useMediaSearchMonitor(props)
  const isDefault = monitor.effectiveState.type === 'default'

  return (
    <div className={cn('relative', monitor.variant === 'pill' && 'w-full', className)}>
      <div className={cn('flex items-center', GAP[monitor.variant], !isDefault && 'invisible')}>
        <DefaultButtons
          variant={monitor.variant}
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
            variant={monitor.variant}
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
  variant: ControlVariant
  theme: MediaTheme
  downloadProgress: ReturnType<typeof useMediaSearchMonitor>['downloadProgress']
  onCompletionClick: () => void
  onErrorDismiss: () => void
}

function StateOverlay({ state, variant, theme, downloadProgress, onCompletionClick, onErrorDismiss }: StateOverlayProps) {
  if (state.type === 'searching') {
    return <SearchingState variant={variant} theme={theme} mode={state.mode} />
  }
  if (state.type === 'progress') {
    return (
      <ProgressState
        variant={variant}
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
    return <CompletedState variant={variant} theme={theme} onClick={onCompletionClick} />
  }
  if (state.type === 'error') {
    return <ErrorState variant={variant} theme={theme} message={state.message} onClick={onErrorDismiss} />
  }
  return null
}
