import { AlertCircle, CornerDownRight, FileVideo, Import, Pencil } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import type { ScannedFile } from '@/types'

import { formatFileSize } from './format-file-size'

function MatchedFileActions({ file, onEditMatch, onImportFile }: {
  file: ScannedFile
  onEditMatch: (f: ScannedFile) => void
  onImportFile: (f: ScannedFile) => void
}) {
  const match = file.suggestedMatch
  if (!match) {
    return null
  }

  return (
    <div className="mt-2 ml-6 flex items-center gap-2">
      <CornerDownRight className="text-muted-foreground size-4 shrink-0" />
      <div className="flex flex-wrap items-center gap-1.5">
        {match.mediaType === 'episode' ? (
          <>
            <Badge className="bg-primary">{match.seriesTitle ?? 'Unknown Series'}</Badge>
            <Badge variant="secondary">
              S{String(match.seasonNum ?? 0).padStart(2, '0')}E
              {String(match.episodeNum ?? 0).padStart(2, '0')} - &quot;{match.mediaTitle}&quot;
            </Badge>
          </>
        ) : (
          <>
            <Badge variant="outline">{match.mediaTitle}</Badge>
            {match.year ? <Badge variant="secondary">{match.year}</Badge> : null}
          </>
        )}
      </div>
      <div className="ml-auto flex shrink-0 gap-1">
        <Button size="sm" variant="ghost" className="h-6 px-2" onClick={() => onEditMatch(file)}>
          <Pencil className="mr-1 size-3" />
          Edit Match
        </Button>
        <Button size="sm" variant="outline" className="h-6 px-2" onClick={() => onImportFile(file)}>
          <Import className="mr-1 size-3" />
          Import
        </Button>
      </div>
    </div>
  )
}

function UnmatchedFileActions({ file, onEditMatch }: {
  file: ScannedFile
  onEditMatch: (f: ScannedFile) => void
}) {
  return (
    <div className="mt-2 ml-6 flex items-center gap-2">
      <CornerDownRight className="text-muted-foreground size-4 shrink-0" />
      <span className="text-muted-foreground text-sm italic">No library match found</span>
      <Button
        size="sm"
        variant="outline"
        className="ml-auto h-6 shrink-0 px-2"
        onClick={() => onEditMatch(file)}
      >
        <Pencil className="mr-1 size-3" />
        Set Match
      </Button>
    </div>
  )
}

function FileHeader({ file, hasMatch }: { file: ScannedFile; hasMatch: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <FileVideo className={`size-4 shrink-0 ${hasMatch ? 'text-blue-600' : 'text-muted-foreground'}`} />
      <span className="truncate text-sm font-medium" title={file.fileName}>{file.fileName}</span>
      <span className="text-muted-foreground shrink-0 text-xs">{formatFileSize(file.fileSize)}</span>
      {!file.valid && (
        <Badge variant="destructive" className="shrink-0 text-xs">
          <AlertCircle className="mr-1 size-3" />
          Invalid
        </Badge>
      )}
    </div>
  )
}

export function ScannedFileRow({ file, isSelected, onToggleFile, onEditMatch, onImportFile }: {
  file: ScannedFile
  isSelected: boolean
  onToggleFile: (path: string) => void
  onEditMatch: (f: ScannedFile) => void
  onImportFile: (f: ScannedFile) => void
}) {
  const hasMatch = !!file.suggestedMatch

  return (
    <div className={`rounded-lg border p-3 ${hasMatch ? 'border-green-600' : ''}`}>
      <div className="flex items-start gap-3">
        <Checkbox checked={isSelected} onCheckedChange={() => onToggleFile(file.path)} disabled={!hasMatch} className="mt-1" />
        <div className="min-w-0 flex-1">
          <FileHeader file={file} hasMatch={hasMatch} />
          {hasMatch ? (
            <MatchedFileActions file={file} onEditMatch={onEditMatch} onImportFile={onImportFile} />
          ) : (
            <UnmatchedFileActions file={file} onEditMatch={onEditMatch} />
          )}
        </div>
      </div>
    </div>
  )
}
