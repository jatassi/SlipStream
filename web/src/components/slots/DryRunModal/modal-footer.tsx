import { Layers } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'

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
    <DialogFooter className="mt-2 shrink-0">
      <DebugFooter
        developerMode={props.developerMode}
        isDebugData={props.isDebugData}
        isLoadingDebugData={props.isLoadingDebugData}
        isExecuting={props.isExecuting}
        onLoadDebugData={props.onLoadDebugData}
      />
      <Button variant="outline" onClick={props.onCancel} disabled={props.isExecuting}>
        Cancel
      </Button>
      <Button onClick={props.onEnable} disabled={!props.canEnable}>
        <Layers className="mr-2 size-4" />
        Enable Multi-Version Mode
      </Button>
    </DialogFooter>
  )
}
