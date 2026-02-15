import { Checkbox } from '@/components/ui/checkbox'
import type { ScannedFile } from '@/types'

import { ScannedFileRow } from './scanned-file-row'

export function ScannedFilesList({
  files,
  selectedFiles,
  onToggleFile,
  onToggleAll,
  onEditMatch,
  onImportFile,
}: {
  files: ScannedFile[]
  selectedFiles: Set<string>
  onToggleFile: (path: string) => void
  onToggleAll: () => void
  onEditMatch: (file: ScannedFile) => void
  onImportFile: (file: ScannedFile) => void
}) {
  const matchedFiles = files.filter((f) => f.suggestedMatch)
  const allSelected =
    matchedFiles.length > 0 && matchedFiles.every((f) => selectedFiles.has(f.path))

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2 border-b px-2 py-1.5">
        <Checkbox checked={allSelected} onCheckedChange={onToggleAll} />
        <span className="text-muted-foreground text-xs">Select all matched files</span>
      </div>

      {files.map((file) => (
        <ScannedFileRow
          key={file.path}
          file={file}
          isSelected={selectedFiles.has(file.path)}
          onToggleFile={onToggleFile}
          onEditMatch={onEditMatch}
          onImportFile={onImportFile}
        />
      ))}
    </div>
  )
}
