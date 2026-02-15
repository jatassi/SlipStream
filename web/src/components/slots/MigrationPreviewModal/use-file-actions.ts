import { useCallback, useMemo, useState } from 'react'

import type { ManualEdit } from './types'

function applyEditsToMap(
  prev: Map<number, ManualEdit>,
  fileIds: Set<number>,
  edit: ManualEdit,
): Map<number, ManualEdit> {
  const next = new Map(prev)
  fileIds.forEach((fileId) => next.set(fileId, edit))
  return next
}

function toggleInSet(prev: Set<number>, fileId: number): Set<number> {
  const next = new Set(prev)
  if (next.has(fileId)) { next.delete(fileId) } else { next.add(fileId) }
  return next
}

function computeIgnoredIds(edits: Map<number, ManualEdit>): Set<number> {
  const ignored = new Set<number>()
  edits.forEach((edit, fileId) => {
    if (edit.type === 'ignore') { ignored.add(fileId) }
  })
  return ignored
}

export function useFileActions(open: boolean) {
  const [selectedFileIds, setSelectedFileIds] = useState<Set<number>>(new Set())
  const [manualEdits, setManualEdits] = useState<Map<number, ManualEdit>>(new Map())
  const [assignModalOpen, setAssignModalOpen] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (!open) {
      setSelectedFileIds(new Set())
      setManualEdits(new Map())
      setAssignModalOpen(false)
    }
  }

  const ignoredFileIds = useMemo(() => computeIgnoredIds(manualEdits), [manualEdits])

  const applyEdit = useCallback((edit: ManualEdit) => {
    setManualEdits((prev) => applyEditsToMap(prev, selectedFileIds, edit))
    setSelectedFileIds(new Set())
  }, [selectedFileIds])

  const handleAssign = useCallback((slotId: number, slotName: string) => {
    applyEdit({ type: 'assign', slotId, slotName })
    setAssignModalOpen(false)
  }, [applyEdit])

  return {
    selectedFileIds, setSelectedFileIds, manualEdits, ignoredFileIds,
    assignModalOpen, setAssignModalOpen,
    handleIgnore: useCallback(() => applyEdit({ type: 'ignore' }), [applyEdit]),
    handleUnassign: useCallback(() => applyEdit({ type: 'unassign' }), [applyEdit]),
    handleAssign,
    handleReset: useCallback(() => { setManualEdits(new Map()); setSelectedFileIds(new Set()) }, []),
    handleToggleFileSelection: useCallback((fileId: number) => {
      setSelectedFileIds((prev) => toggleInSet(prev, fileId))
    }, []),
  }
}
