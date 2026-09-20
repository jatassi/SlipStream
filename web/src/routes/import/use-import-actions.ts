import { useState } from 'react'

import { toast } from 'sonner'

import { useManualImport } from '@/hooks'
import type { ManualImportRequest, ScannedFile, SuggestedMatch } from '@/types'

export type MatchParams = {
  mediaType: string
  mediaId: number
  seriesId?: number
  seasonNum?: number
  targetSlotId?: number
}

function matchToRequest(path: string, match: MatchParams): ManualImportRequest {
  return {
    path,
    mediaType: match.mediaType as 'movie' | 'episode',
    mediaId: match.mediaId,
    seriesId: match.seriesId,
    seasonNum: match.seasonNum,
    targetSlotId: match.targetSlotId,
  }
}

function suggestedToRequest(file: ScannedFile, match: SuggestedMatch): ManualImportRequest {
  return {
    path: file.path,
    mediaType: match.mediaType as 'movie' | 'episode',
    mediaId: match.mediaId,
    seriesId: match.seriesId,
    seasonNum: match.seasonNum,
  }
}

type Importer = (file: ScannedFile, request: ManualImportRequest) => Promise<boolean>

function useImporter(onImported: (path: string) => void): {
  run: Importer
  isImporting: boolean
} {
  const mutation = useManualImport()
  return {
    isImporting: mutation.isPending,
    run: async (file, request) => {
      try {
        const result = await mutation.mutateAsync(request)
        if (!result.success) {
          toast.error(result.error ?? `Failed to import ${file.fileName}`)
          return false
        }
        onImported(file.path)
        return true
      } catch {
        toast.error(`Failed to import ${file.fileName}`)
        return false
      }
    },
  }
}

async function importSuggested(run: Importer, file: ScannedFile): Promise<boolean> {
  if (!file.suggestedMatch) {
    return false
  }
  return run(file, suggestedToRequest(file, file.suggestedMatch))
}

export function useImportActions() {
  const [imported, setImported] = useState<Set<string>>(new Set())
  const [editing, setEditing] = useState<ScannedFile | null>(null)
  const { run, isImporting } = useImporter((path) => {
    setImported((prev) => new Set(prev).add(path))
  })

  return {
    imported,
    editing,
    setEditing,
    isImporting,
    importFile: async (file: ScannedFile) => {
      if (await importSuggested(run, file)) {
        toast.success(`Imported ${file.fileName}`)
      }
    },
    importAll: async (files: ScannedFile[]) => {
      let count = 0
      for (const file of files) {
        if (await importSuggested(run, file)) {
          count += 1
        }
      }
      if (count > 0) {
        toast.success(`Imported ${count.toString()} file${count === 1 ? '' : 's'}`)
      }
    },
    confirmMatch: async (file: ScannedFile, match: MatchParams) => {
      setEditing(null)
      if (await run(file, matchToRequest(file.path, match))) {
        toast.success(`Imported ${file.fileName}`)
      }
    },
  }
}
