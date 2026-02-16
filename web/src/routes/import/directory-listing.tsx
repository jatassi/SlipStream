import { ChevronRight, ChevronUp, FileVideo, FolderOpen, HardDrive } from 'lucide-react'

import { ScrollArea } from '@/components/ui/scroll-area'

import { formatFileSize } from './format-file-size'

type DirectoryEntry = { name: string; path: string }
type FileEntry = { name: string; path: string; size: number }
type DriveEntry = { letter: string; label?: string }

export type BrowseData = {
  parent?: string
  drives?: DriveEntry[]
  directories: DirectoryEntry[]
  files: FileEntry[]
}

function BackButton({ onNavigateUp }: { onNavigateUp: () => void }) {
  return (
    <button
      onClick={onNavigateUp}
      className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
    >
      <ChevronUp className="text-muted-foreground size-4" />
      <span className="text-sm">..</span>
    </button>
  )
}

function DriveItem({ drive, onNavigateTo }: { drive: DriveEntry; onNavigateTo: (p: string) => void }) {
  return (
    <button
      onClick={() => onNavigateTo(`${drive.letter}\\`)}
      className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
    >
      <HardDrive className="text-muted-foreground size-4" />
      <span className="text-sm font-medium">{drive.letter}</span>
      {drive.label ? <span className="text-muted-foreground text-sm">({drive.label})</span> : null}
    </button>
  )
}

function DirectoryItem({ dir, onNavigateTo }: { dir: DirectoryEntry; onNavigateTo: (p: string) => void }) {
  return (
    <button
      onClick={() => onNavigateTo(dir.path)}
      className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
    >
      <FolderOpen className="size-4 text-yellow-600" />
      <span className="text-sm">{dir.name}</span>
      <ChevronRight className="text-muted-foreground ml-auto size-4" />
    </button>
  )
}

function FileItem({ file }: { file: FileEntry }) {
  return (
    <div className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2">
      <FileVideo className="size-4 text-blue-600" />
      <span className="flex-1 truncate text-sm">{file.name}</span>
      <span className="text-muted-foreground text-xs">{formatFileSize(file.size)}</span>
    </div>
  )
}

function EmptyState({ show }: { show: boolean }) {
  if (!show) {
    return null
  }
  return <p className="text-muted-foreground py-4 text-center text-sm">No items found</p>
}

function ListingItems({ data, currentPath, onNavigateTo, onNavigateUp }: {
  data: BrowseData | undefined
  currentPath: string
  onNavigateTo: (path: string) => void
  onNavigateUp: () => void
}) {
  const showBack = Boolean(currentPath || data?.parent)
  const drives = data?.drives ?? []
  const dirs = data?.directories ?? []
  const files = data?.files ?? []

  return (
    <>
      {showBack ? <BackButton onNavigateUp={onNavigateUp} /> : null}
      {drives.map((d) => <DriveItem key={d.letter} drive={d} onNavigateTo={onNavigateTo} />)}
      {dirs.map((d) => <DirectoryItem key={d.path} dir={d} onNavigateTo={onNavigateTo} />)}
      {files.map((f) => <FileItem key={f.path} file={f} />)}
    </>
  )
}

export function DirectoryListing(props: {
  data: BrowseData | undefined
  currentPath: string
  onNavigateTo: (path: string) => void
  onNavigateUp: () => void
}) {
  const { data } = props
  const isEmpty = data && (data.drives?.length ?? 0) === 0 && (data.directories?.length ?? 0) === 0 && (data.files?.length ?? 0) === 0

  return (
    <ScrollArea className="h-[400px]">
      <div className="space-y-1">
        <ListingItems {...props} />
        <EmptyState show={!!isEmpty} />
      </div>
    </ScrollArea>
  )
}
