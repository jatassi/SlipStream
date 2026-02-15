import { useState } from 'react'

import type { ScannedFile } from '@/types'

export function useFileSelection(scannedFiles: ScannedFile[]) {
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set())

  const toggleFile = (path: string) => {
    const file = scannedFiles.find((f) => f.path === path)
    if (!file?.suggestedMatch) {
      return
    }

    setSelectedFiles((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const toggleAll = () => {
    const matchedFiles = scannedFiles.filter((f) => f.suggestedMatch)
    const allSelected =
      matchedFiles.length > 0 && matchedFiles.every((f) => selectedFiles.has(f.path))

    if (allSelected) {
      setSelectedFiles(new Set())
    } else {
      setSelectedFiles(new Set(matchedFiles.map((f) => f.path)))
    }
  }

  return { selectedFiles, setSelectedFiles, toggleFile, toggleAll }
}
