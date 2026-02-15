import { ChevronRight, Folder, FolderUp, HardDrive } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { formatBytes } from '@/lib/formatters'

export function PathInput({
  inputPath,
  setInputPath,
  hasDrives,
  onSubmit,
}: {
  inputPath: string
  setInputPath: (path: string) => void
  hasDrives: boolean
  onSubmit: (e: React.FormEvent) => void
}) {
  return (
    <form onSubmit={onSubmit} className="flex gap-2">
      <input
        value={inputPath}
        onChange={(e) => setInputPath(e.target.value)}
        placeholder={hasDrives ? 'Select a drive or enter path...' : '/path/to/folder'}
        className="font-mono text-sm flex-1 border border-border rounded px-3 py-2 bg-background"
      />
      <Button type="submit" variant="outline" size="sm">
        Go
      </Button>
    </form>
  )
}

export function SelectedPath({ path }: { path: string }) {
  return (
    <div className="text-sm">
      <span className="text-muted-foreground">Selected: </span>
      <span className="font-mono">{path}</span>
    </div>
  )
}

export function Breadcrumbs({
  breadcrumbs,
  onNavigate,
}: {
  breadcrumbs: { label: string; path: string }[]
  onNavigate: (path: string) => void
}) {
  return (
    <div className="flex items-center gap-1 overflow-x-auto pb-1 text-sm">
      <Button variant="ghost" size="sm" className="h-7 px-2" onClick={() => onNavigate('')}>
        <HardDrive className="size-4" />
      </Button>
      {breadcrumbs.map((crumb, index) => (
        <div key={crumb.path} className="flex items-center">
          <ChevronRight className="text-muted-foreground size-4" />
          <Button
            variant="ghost"
            size="sm"
            className={`h-7 px-2 ${index === breadcrumbs.length - 1 ? 'font-medium' : ''}`}
            onClick={() => onNavigate(crumb.path)}
          >
            {crumb.label}
          </Button>
        </div>
      ))}
    </div>
  )
}

export function DrivesList({
  drives,
  onNavigate,
}: {
  drives: { letter: string; label?: string; freeSpace?: number }[]
  onNavigate: (path: string) => void
}) {
  return (
    <div className="p-2">
      <p className="text-muted-foreground mb-2 px-2 text-xs font-medium">Drives</p>
      {drives.map((drive) => (
        <button
          key={drive.letter}
          className="hover:bg-accent flex w-full items-center gap-3 rounded-md px-3 py-2 text-left"
          onClick={() => onNavigate(`${drive.letter}\\`)}
        >
          <HardDrive className="text-muted-foreground size-5" />
          <div className="flex-1">
            <span className="font-medium">{drive.letter}</span>
            {drive.label ? (
              <span className="text-muted-foreground ml-2">{drive.label}</span>
            ) : null}
          </div>
          {drive.freeSpace !== undefined && drive.freeSpace > 0 && (
            <span className="text-muted-foreground text-xs">
              {formatBytes(drive.freeSpace)} free
            </span>
          )}
        </button>
      ))}
    </div>
  )
}

export function ParentButton({
  parent,
  onNavigate,
}: {
  parent: string
  onNavigate: (path: string) => void
}) {
  return (
    <button
      className="hover:bg-accent flex w-full items-center gap-3 border-b px-3 py-2 text-left"
      onClick={() => onNavigate(parent)}
    >
      <FolderUp className="text-muted-foreground size-5" />
      <span className="text-muted-foreground">..</span>
    </button>
  )
}

export function EntriesList({
  entries,
  onNavigate,
}: {
  entries: { path: string; name: string }[]
  onNavigate: (path: string) => void
}) {
  return (
    <div className="p-1">
      {entries.map((entry) => (
        <button
          key={entry.path}
          className="hover:bg-accent flex w-full items-center gap-3 rounded-md px-3 py-2 text-left"
          onClick={() => onNavigate(entry.path)}
        >
          <Folder className="size-5 text-blue-500" />
          <span className="truncate">{entry.name}</span>
        </button>
      ))}
    </div>
  )
}

export function EmptyMessage() {
  return (
    <div className="text-muted-foreground flex h-32 items-center justify-center">
      No subdirectories
    </div>
  )
}
