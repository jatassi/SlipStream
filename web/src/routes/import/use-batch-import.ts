import { useCallback } from 'react'

import { toast } from 'sonner'

import type { ScannedFile } from '@/types'

import { suggestedToRequest } from './use-import-page'

type ImportMutation = {
  mutateAsync: (req: { path: string; mediaType: 'movie' | 'episode'; mediaId: number; seriesId?: number; seasonNum?: number }) =>
    Promise<{ success: boolean; error?: string }>
}

function showBatchResult(successCount: number, failCount: number, lastError: string) {
  if (successCount > 0 && failCount === 0) {
    toast.success(`Imported ${successCount} file${successCount > 1 ? 's' : ''}`)
  } else if (successCount > 0) {
    toast.warning(`Imported ${successCount}, failed ${failCount}`)
  } else {
    toast.error(lastError || 'Failed to import files')
  }
}

async function importOneFile(
  file: ScannedFile,
  mutation: ImportMutation,
  removeFile: (path: string) => void,
): Promise<{ success: boolean; error: string }> {
  if (!file.suggestedMatch) {
    return { success: false, error: '' }
  }
  try {
    const result = await mutation.mutateAsync(suggestedToRequest(file.path, file.suggestedMatch))
    if (result.success) {
      removeFile(file.path)
      return { success: true, error: '' }
    }
    return { success: false, error: result.error ?? '' }
  } catch {
    return { success: false, error: '' }
  }
}

export function useBatchImport({
  scannedFiles,
  selectedFiles,
  importMutation,
  removeFileFromScan,
}: {
  scannedFiles: ScannedFile[]
  selectedFiles: Set<string>
  importMutation: ImportMutation
  removeFileFromScan: (path: string) => void
}) {
  const handleImportSelected = useCallback(async () => {
    const filesToImport = scannedFiles.filter((f) => selectedFiles.has(f.path) && f.suggestedMatch)
    if (filesToImport.length === 0) {
      return
    }

    let successCount = 0
    let failCount = 0
    let lastError = ''

    for (const file of filesToImport) {
      const outcome = await importOneFile(file, importMutation, removeFileFromScan)
      if (outcome.success) {
        successCount++
      } else {
        failCount++
        lastError = outcome.error || lastError
      }
    }

    showBatchResult(successCount, failCount, lastError)
  }, [scannedFiles, selectedFiles, importMutation, removeFileFromScan])

  return { handleImportSelected }
}
