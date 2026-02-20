import { useCallback } from 'react'

import { toast } from 'sonner'

import { useExecuteMigration, useSlots } from '@/hooks'

import { generateDebugPreview } from '../shared/debug'
import type { DryRunModalProps, ManualEdit } from '../shared/types'
import type { ModalStateBundle } from './use-modal-state'

type SetState = React.Dispatch<React.SetStateAction<ModalStateBundle>>
type Patch = (u: Partial<ModalStateBundle>) => void

type ActionDeps = {
  setS: SetState
  patch: Patch
  props: DryRunModalProps
  visibleFileIds: number[]
  selectedFileIds: Set<number>
  manualEdits: Map<number, ManualEdit>
}

type ExecuteDeps = {
  manualEdits: Map<number, ManualEdit>
  executeMutation: ReturnType<typeof useExecuteMigration>
  patch: Patch
  props: DryRunModalProps
}

function toggleSelectAll(setS: SetState, visibleFileIds: number[], selectedFileIds: Set<number>) {
  const allSelected = visibleFileIds.every((id) => selectedFileIds.has(id))
  setS((prev) => {
    const next = new Set(prev.selectedFileIds)
    for (const id of visibleFileIds) {
      if (allSelected) { next.delete(id) } else { next.add(id) }
    }
    return { ...prev, selectedFileIds: next }
  })
}

function bulkEdit(setS: SetState, edit: ManualEdit) {
  setS((prev) => {
    const next = new Map(prev.manualEdits)
    prev.selectedFileIds.forEach((fileId) => { next.set(fileId, edit) })
    return { ...prev, manualEdits: next, selectedFileIds: new Set() }
  })
}

function toggleFile(setS: SetState, fileId: number) {
  setS((prev) => {
    const next = new Set(prev.selectedFileIds)
    if (next.has(fileId)) { next.delete(fileId) } else { next.add(fileId) }
    return { ...prev, selectedFileIds: next }
  })
}

export function useModalActions(deps: ActionDeps) {
  const { setS, patch, props, visibleFileIds, selectedFileIds, manualEdits } = deps
  const executeMutation = useExecuteMigration()
  const { data: slots = [] } = useSlots()

  const handleToggleSelectAll = useCallback(
    () => { toggleSelectAll(setS, visibleFileIds, selectedFileIds) },
    [setS, visibleFileIds, selectedFileIds],
  )
  const handleToggleFileSelection = useCallback(
    (fileId: number) => { toggleFile(setS, fileId) },
    [setS],
  )
  const handleIgnore = useCallback(() => { bulkEdit(setS, { type: 'ignore' }) }, [setS])
  const handleUnassign = useCallback(() => { bulkEdit(setS, { type: 'unassign' }) }, [setS])
  const handleAssign = useCallback(
    (slotId: number, slotName: string) => {
      bulkEdit(setS, { type: 'assign', slotId, slotName })
      patch({ assignModalOpen: false })
    },
    [setS, patch],
  )
  const handleReset = useCallback(
    () => { patch({ manualEdits: new Map(), selectedFileIds: new Set() }) },
    [patch],
  )

  const handleLoadDebugData = useLoadDebugData(patch)
  const handleExecute = useExecuteHandler({ manualEdits, executeMutation, patch, props })

  return {
    slots,
    isExecuting: executeMutation.isPending,
    handleToggleSelectAll,
    handleToggleFileSelection,
    handleIgnore,
    handleUnassign,
    handleAssign,
    handleReset,
    handleLoadDebugData,
    handleExecute,
  }
}

function useLoadDebugData(patch: Patch) {
  return useCallback(async () => {
    patch({ isLoadingDebugData: true })
    try {
      const debugPreview = await generateDebugPreview()
      patch({
        preview: debugPreview,
        isDebugData: true,
        isLoadingDebugData: false,
        activeTab: 'movies',
        filter: 'all',
        selectedFileIds: new Set(),
        manualEdits: new Map(),
      })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to generate debug data')
      patch({ isLoadingDebugData: false })
    }
  }, [patch])
}

function useExecuteHandler({ manualEdits, executeMutation, patch, props }: ExecuteDeps) {
  const { onOpenChange, onMigrationComplete, onMigrationFailed } = props

  return useCallback(() => {
    const overrides = [...manualEdits.entries()].map(([fileId, edit]) => ({
      fileId,
      type: edit.type,
      slotId: edit.type === 'assign' ? edit.slotId : undefined,
    }))

    const closeAndReport = (errorMessage?: string) => {
      patch({ confirmModalOpen: false })
      onOpenChange(false)
      if (errorMessage) { onMigrationFailed?.(errorMessage) }
      else { onMigrationComplete() }
    }

    executeMutation.mutate(overrides.length > 0 ? { overrides } : undefined, {
      onSuccess: (result) => {
        if (result.success) {
          toast.success(`Multi-version mode enabled! ${result.filesAssigned} files assigned to slots.`)
          closeAndReport()
        } else {
          closeAndReport(result.errors.join('. ') || 'Migration completed with errors')
        }
      },
      onError: (error) => {
        closeAndReport(error instanceof Error ? error.message : 'Migration failed')
      },
    })
  }, [manualEdits, executeMutation, patch, onOpenChange, onMigrationComplete, onMigrationFailed])
}
