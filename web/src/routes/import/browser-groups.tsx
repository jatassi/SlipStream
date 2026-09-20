import { FileVideo, Folder } from 'lucide-react'

import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import type { ScannedFile } from '@/types'

import { formatFileSize } from './format-file-size'
import { importableFile, matchLabel } from './match-label'
import type { BrowserFile, BrowserFolder } from './use-import-browser'

const SKELETONS = ['browse-a', 'browse-b', 'browse-c', 'browse-d'] as const

const ROW = 'flex min-h-tap w-full items-center gap-3 px-4 py-1.5'
const LABEL =
  'min-h-tap focus-visible:ring-ring flex min-w-0 flex-1 flex-col justify-center text-left outline-none focus-visible:ring-[3px]'
const ACTION =
  'press min-h-tap focus-visible:ring-ring text-footnote text-primary shrink-0 rounded-[10px] px-3 font-semibold outline-none focus-visible:ring-[3px]'

export function BrowserSkeleton() {
  return (
    <Group>
      {SKELETONS.map((id) => (
        <RowSkeleton key={id} subtitle={false} chevron />
      ))}
    </Group>
  )
}

export function FolderGroup({
  folders,
  onOpen,
}: {
  folders: BrowserFolder[]
  onOpen: (path: string) => void
}) {
  if (folders.length === 0) {
    return null
  }
  return (
    <Group header="Folders">
      {folders.map((folder) => (
        <Row
          key={folder.path}
          title={folder.name}
          chevron
          leading={
            <IconTile className="bg-amber-500">
              <Folder />
            </IconTile>
          }
          onClick={() => {
            onOpen(folder.path)
          }}
        />
      ))}
    </Group>
  )
}

function FileLabel({
  file,
  scanned,
  matching,
  onEdit,
}: {
  file: BrowserFile
  scanned: ScannedFile | undefined
  matching: boolean
  onEdit: (scanned: ScannedFile) => void
}) {
  const detail = `${matchLabel(scanned, matching)} · ${formatFileSize(file.size)}`
  if (scanned === undefined) {
    return (
      <div className={LABEL}>
        <div className="text-footnote truncate font-mono">{file.name}</div>
        <div className="text-footnote text-muted-foreground mt-0.5 truncate">{detail}</div>
      </div>
    )
  }
  return (
    <button
      type="button"
      aria-label={file.name}
      className={LABEL}
      onClick={() => {
        onEdit(scanned)
      }}
    >
      <span className="text-footnote block truncate font-mono">{file.name}</span>
      <span className="text-footnote text-muted-foreground mt-0.5 block truncate">{detail}</span>
    </button>
  )
}

function ImportAction({
  file,
  scanned,
  disabled,
  onImport,
}: {
  file: BrowserFile
  scanned: ScannedFile | undefined
  disabled: boolean
  onImport: (scanned: ScannedFile) => void
}) {
  const importable = importableFile(scanned)
  if (importable === undefined) {
    return null
  }
  return (
    <button
      type="button"
      aria-label={`Import ${file.name}`}
      disabled={disabled}
      className={ACTION}
      onClick={() => {
        onImport(importable)
      }}
    >
      Import
    </button>
  )
}

type FileGroupProps = {
  files: BrowserFile[]
  matches: Map<string, ScannedFile>
  matching: boolean
  isImporting: boolean
  onImport: (scanned: ScannedFile) => void
  onEdit: (scanned: ScannedFile) => void
}

export function FileGroup({
  files,
  matches,
  matching,
  isImporting,
  onImport,
  onEdit,
}: FileGroupProps) {
  if (files.length === 0) {
    return null
  }
  return (
    <Group header="Files">
      {files.map((file) => (
        <div key={file.path} className={ROW}>
          <IconTile className="bg-muted-foreground">
            <FileVideo />
          </IconTile>
          <FileLabel
            file={file}
            scanned={matches.get(file.path)}
            matching={matching}
            onEdit={onEdit}
          />
          <ImportAction
            file={file}
            scanned={matches.get(file.path)}
            disabled={isImporting}
            onImport={onImport}
          />
        </div>
      ))}
    </Group>
  )
}
