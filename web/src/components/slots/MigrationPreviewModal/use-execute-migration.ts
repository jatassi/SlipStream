import { useCallback } from 'react'

import { toast } from 'sonner'

import { useExecuteMigration } from '@/hooks'

import type { ManualEdit } from './types'

export function useExecuteHandler(
  manualEdits: Map<number, ManualEdit>,
  onOpenChange: (open: boolean) => void,
  onMigrationComplete: () => void,
) {
  const executeMutation = useExecuteMigration()

  const handleExecute = useCallback(() => {
    const overrides = [...manualEdits.entries()].map(([fileId, edit]) => ({
      fileId,
      type: edit.type,
      slotId: edit.type === 'assign' ? edit.slotId : undefined,
    }))

    executeMutation.mutate(overrides.length > 0 ? { overrides } : undefined, {
      onSuccess: (result) => {
        if (result.success) {
          toast.success(`Migration complete: ${result.filesAssigned} files assigned`)
          onOpenChange(false)
          onMigrationComplete()
        } else {
          toast.error('Migration completed with errors')
        }
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Migration failed')
      },
    })
  }, [manualEdits, executeMutation, onOpenChange, onMigrationComplete])

  return {
    handleExecute,
    isExecuting: executeMutation.isPending,
  }
}
