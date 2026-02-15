import { useCallback, useState } from 'react'

import { toast } from 'sonner'

import { useManualImport, useScanDirectory } from '@/hooks/use-import'
import type { ManualImportRequest, ScannedFile, SuggestedMatch } from '@/types'

import { useBatchImport } from './use-batch-import'
import { useFileSelection } from './use-file-selection'

function removeFromSet(set: Set<string>, key: string): Set<string> {
  const next = new Set(set)
  next.delete(key)
  return next
}

export type MatchParams = {
  mediaType: string
  mediaId: number
  seriesId?: number
  seasonNum?: number
  targetSlotId?: number
}

export function matchToRequest(path: string, match: MatchParams): ManualImportRequest {
  return {
    path,
    mediaType: match.mediaType as 'movie' | 'episode',
    mediaId: match.mediaId,
    seriesId: match.seriesId,
    seasonNum: match.seasonNum,
    targetSlotId: match.targetSlotId,
  }
}

export function suggestedToRequest(path: string, match: SuggestedMatch): ManualImportRequest {
  return {
    path,
    mediaType: match.mediaType as 'movie' | 'episode',
    mediaId: match.mediaId,
    seriesId: match.seriesId,
    seasonNum: match.seasonNum,
  }
}

export function useImportPage() {
  const [currentPath, setCurrentPath] = useState('')
  const [scannedFiles, setScannedFiles] = useState<ScannedFile[]>([])
  const [dialogFile, setDialogFile] = useState<ScannedFile | null>(null)
  const scanMutation = useScanDirectory()
  const importMutation = useManualImport()
  const { selectedFiles, setSelectedFiles, toggleFile, toggleAll } = useFileSelection(scannedFiles)

  const removeFile = (filePath: string) => {
    setScannedFiles((prev) => prev.filter((f) => f.path !== filePath))
    setSelectedFiles((prev) => removeFromSet(prev, filePath))
  }

  const { handleImportSelected } = useBatchImport({ scannedFiles, selectedFiles, importMutation, removeFileFromScan: removeFile })

  const handleScanPath = useCallback(async (path: string) => {
    try {
      const result = await scanMutation.mutateAsync({ path })
      setScannedFiles(result.files)
      setSelectedFiles(new Set(result.files.filter((f) => f.suggestedMatch).map((f) => f.path)))
    } catch {
      toast.error('Failed to scan directory')
    }
  }, [scanMutation, setSelectedFiles])

  const handleConfirmImport = async (file: ScannedFile, match: MatchParams) => {
    try {
      const result = await importMutation.mutateAsync(matchToRequest(file.path, match))
      if (result.success) { toast.success(`Imported ${file.fileName}`); removeFile(file.path) }
      else { toast.error(result.error ?? 'Import failed') }
    } catch { toast.error('Failed to import file') }
    setDialogFile(null)
  }

  const handleDirectImport = async (file: ScannedFile) => {
    if (!file.suggestedMatch) { return }
    try {
      const result = await importMutation.mutateAsync(suggestedToRequest(file.path, file.suggestedMatch))
      if (result.success) { toast.success(`Imported ${file.fileName}`); removeFile(file.path) }
      else { toast.error(result.error ?? 'Import failed') }
    } catch { toast.error('Failed to import file') }
  }

  return {
    currentPath, setCurrentPath, scannedFiles, selectedFiles, importDialogFile: dialogFile,
    isScanning: scanMutation.isPending, isImporting: importMutation.isPending,
    handleScanPath, handleToggleFile: toggleFile, handleToggleAll: toggleAll,
    handleImportSelected, handleEditMatch: setDialogFile, handleConfirmImport,
    handleClearScan: () => { setScannedFiles([]); setSelectedFiles(new Set()) },
    handleDirectImport, closeDialog: () => setDialogFile(null),
  }
}
