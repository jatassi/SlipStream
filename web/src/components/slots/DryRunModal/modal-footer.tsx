import { Layers } from 'lucide-react'

import { Button } from '@/components/ui/button'

import { DebugFooter } from './debug-footer'

type ModalFooterProps = {
  developerMode: boolean
  isDebugData: boolean
  isLoadingDebugData: boolean
  isLoading: boolean
  isExecuting: boolean
  canEnable: boolean
  onLoadDebugData: () => void
  onCancel: () => void
  onEnable: () => void
}

export function ModalFooter(props: ModalFooterProps) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end">
      <DebugFooter
        developerMode={props.developerMode}
        isDebugData={props.isDebugData}
        isLoadingDebugData={props.isLoadingDebugData}
        isExecuting={props.isExecuting}
        onLoadDebugData={props.onLoadDebugData}
      />
      <Button
        variant="outline"
        className="min-h-tap"
        onClick={props.onCancel}
        disabled={props.isExecuting}
      >
        Cancel
      </Button>
      <Button className="min-h-tap" onClick={props.onEnable} disabled={!props.canEnable}>
        <Layers className="mr-2 size-4" />
        Enable Multi-Version Mode
      </Button>
    </div>
  )
}
